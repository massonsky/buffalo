# Buffalo Plugin System

## Overview

Buffalo provides a powerful plugin system that allows you to extend functionality without modifying core code. Plugins can add new language support, validate proto files, transform generated code, and hook into various build stages.

## Plugin Types

### 1. Compiler Plugins (`PluginTypeCompiler`)
Add support for new programming languages.

**Example:** TypeScript, Java, Kotlin, Swift compilers

**Interface:**
```go
type CompilerPlugin interface {
    Plugin
    SupportedLanguage() string
    RequiredTools() []string
    ValidateEnvironment() error
}
```

### 2. Validator Plugins (`PluginTypeValidator`)
Validate proto files and generated code.

**Example:** Style checkers, breaking change detectors, API guidelines

**Interface:**
```go
type ValidatorPlugin interface {
    Plugin
    ValidationRules() []string
}
```

### 3. Transformer Plugins (`PluginTypeTransformer`)
Transform or post-process generated code.

**Example:** Code formatters, comment generators, builder pattern adders

**Interface:**
```go
type TransformerPlugin interface {
    Plugin
    SupportedFileTypes() []string
}
```

### 4. Hook Plugins (`PluginTypeHook`)
Execute custom logic at specific build stages.

**Example:** Metrics collectors, notification senders, artifact uploaders

**Interface:**
```go
type HookPlugin interface {
    Plugin
    CanModifyFiles() bool
}
```

### 5. Generator Plugins (`PluginTypeGenerator`)
Generate additional artifacts beyond compiled proto files.

**Example:** OpenAPI/Swagger specs, documentation, client SDKs

## Hook Points

Plugins can execute at different stages of the build process:

| Hook Point | When It Runs | Use Cases |
|------------|--------------|-----------|
| `pre-build` | Before any build operations | Validation, preprocessing |
| `post-parse` | After proto files are parsed | AST analysis, dependency checks |
| `pre-compile` | Before compilation of each language | Environment setup, tool validation |
| `post-compile` | After compilation of each language | Code formatting, patching |
| `post-build` | After all build operations | Documentation, artifact upload |

## Creating a Plugin

### Step 1: Implement the Plugin Interface

```go
package main

import (
    "context"
    "github.com/massonsky/buffalo/internal/plugin"
)

type MyPlugin struct {
    config plugin.Config
}

func New() plugin.Plugin {
    return &MyPlugin{}
}

func (p *MyPlugin) Name() string {
    return "my-plugin"
}

func (p *MyPlugin) Version() string {
    return "1.0.0"
}

func (p *MyPlugin) Type() plugin.PluginType {
    return plugin.PluginTypeValidator
}

func (p *MyPlugin) Description() string {
    return "My custom plugin"
}

func (p *MyPlugin) Init(config plugin.Config) error {
    p.config = config
    return nil
}

func (p *MyPlugin) Execute(ctx context.Context, input *plugin.Input) (*plugin.Output, error) {
    // Your plugin logic here
    return &plugin.Output{
        Success: true,
        Messages: []string{"Plugin executed successfully"},
    }, nil
}

func (p *MyPlugin) Shutdown() error {
    return nil
}
```

### Step 2: Build the Plugin

```bash
# Build as a Go plugin (.so file)
go build -buildmode=plugin -o my-plugin.so my-plugin.go
```

### Step 3: Install the Plugin

```bash
# Copy to Buffalo plugins directory
mkdir -p ~/.buffalo/plugins/my-plugin
cp my-plugin.so ~/.buffalo/plugins/my-plugin/plugin.so
```

### Step 4: Configure the Plugin

Add to your `buffalo.yaml`:

```yaml
plugins:
  - name: my-plugin
    enabled: true
    hooks:
      - pre-build
    priority: 100
    config:
      custom_option: value
```

## Plugin Input/Output

### Input
```go
type Input struct {
    ProtoFiles     []string               // Proto files being processed
    OutputDir      string                 // Base output directory
    Language       string                 // Target language (for compilers)
    ImportPaths    []string               // Proto import paths
    GeneratedFiles []string               // Files from previous stages
    Metadata       map[string]interface{} // Additional context
    WorkingDir     string                 // Current working directory
}
```

### Output
```go
type Output struct {
    Success        bool                   // Execution success status
    GeneratedFiles []string               // New files created
    ModifiedFiles  []string               // Existing files modified
    Messages       []string               // Info messages
    Warnings       []string               // Non-fatal warnings
    Errors         []string               // Fatal errors
    Metadata       map[string]interface{} // Additional output data
}
```

## Example Plugins

### 1. Naming Validator

Validates proto file naming conventions.

**Features:**
- Checks snake_case format
- No spaces in filenames
- Proper .proto extension
- Optional strict mode

**Location:** `internal/plugin/examples/naming_validator.go`

**Configuration:**
```yaml
plugins:
  - name: naming-validator
    enabled: true
    hooks:
      - pre-build
    config:
      strict_mode: false
```

### 2. TypeScript Compiler (Example)

```go
type TypeScriptCompiler struct {
    outputDir string
    npmPath   string
}

func (t *TypeScriptCompiler) SupportedLanguage() string {
    return "typescript"
}

func (t *TypeScriptCompiler) RequiredTools() []string {
    return []string{"protoc", "protoc-gen-ts"}
}

func (t *TypeScriptCompiler) Execute(ctx context.Context, input *plugin.Input) (*plugin.Output, error) {
    for _, protoFile := range input.ProtoFiles {
        cmd := exec.CommandContext(ctx, "protoc",
            "--plugin=protoc-gen-ts="+t.npmPath+"/protoc-gen-ts",
            "--ts_out="+t.outputDir,
            "--proto_path="+input.WorkingDir,
            protoFile)
        // Execute...
    }
    // Return output...
}
```

### 3. Documentation Generator (Example)

```go
func (d *DocGenerator) Execute(ctx context.Context, input *plugin.Input) (*plugin.Output, error) {
    var docs strings.Builder
    
    docs.WriteString("# API Documentation\n\n")
    
    for _, protoFile := range input.ProtoFiles {
        // Parse proto file
        // Generate markdown documentation
        docs.WriteString(fmt.Sprintf("## %s\n\n", protoFile))
        // ...
    }
    
    outputFile := filepath.Join(input.OutputDir, "docs", "API.md")
    if err := os.WriteFile(outputFile, []byte(docs.String()), 0644); err != nil {
        return nil, err
    }
    
    return &plugin.Output{
        Success:        true,
        GeneratedFiles: []string{outputFile},
        Messages:       []string{"Documentation generated"},
    }, nil
}
```

## Best Practices

### 1. Error Handling
- Return descriptive errors
- Don't panic - return errors instead
- Log warnings for non-fatal issues

### 2. Performance
- Minimize file I/O
- Use context for cancellation
- Parallelize when possible

### 3. Configuration
- Provide sensible defaults
- Validate configuration in `Init()`
- Document all config options

### 4. Logging
- Use plugin.Output for messages
- Separate info, warnings, and errors
- Include context in messages

### 5. Testing
- Unit test plugin logic
- Test with various inputs
- Handle edge cases

## Plugin Priority

Plugins execute in priority order (higher = earlier):

- **300+**: Critical validators
- **200-299**: Pre-processors
- **100-199**: Standard plugins (default)
- **50-99**: Post-processors
- **1-49**: Cleanup and finalization

## Troubleshooting

### Plugin Not Loading
1. Check file exists: `~/.buffalo/plugins/[name]/plugin.so`
2. Verify Go version matches Buffalo
3. Check for `New()` function export
4. Review Buffalo logs with `--verbose`

### Plugin Execution Fails
1. Check plugin logs in output
2. Verify required tools are installed
3. Test plugin independently
4. Check hook point configuration

### Configuration Issues
1. Validate YAML syntax
2. Check option names match plugin expectations
3. Ensure hook points are valid
4. Verify priority doesn't conflict

## Advanced Topics

### Plugin Dependencies
Plugins can depend on external tools:

```go
func (p *MyPlugin) ValidateEnvironment() error {
    if _, err := exec.LookPath("my-tool"); err != nil {
        return fmt.Errorf("required tool 'my-tool' not found")
    }
    return nil
}
```

### State Management
Plugins can maintain state between executions:

```go
type StatefulPlugin struct {
    cache map[string]string
    mu    sync.RWMutex
}
```

### Plugin Communication
Plugins can pass data via `Input.Metadata`:

```go
// Plugin 1 sets data
output.Metadata["my_data"] = someValue

// Plugin 2 reads data (next hook point)
if data, ok := input.Metadata["my_data"]; ok {
    // Use data...
}
```

## Contributing Plugins

To share your plugin with the community:

1. Create a GitHub repository
2. Add README with usage instructions
3. Include example configuration
4. Provide build instructions
5. Tag releases with semantic versioning

## License

Plugins are independent works and can use any license. Buffalo itself is licensed under [your license].
