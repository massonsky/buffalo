# Buffalo Plugin CLI Reference

Complete reference for Buffalo plugin management commands.

## Table of Contents

- [Overview](#overview)
- [Commands](#commands)
  - [buffalo plugin list](#buffalo-plugin-list)
  - [buffalo plugin install](#buffalo-plugin-install)
  - [buffalo plugin remove](#buffalo-plugin-remove)
  - [buffalo plugin enable](#buffalo-plugin-enable)
  - [buffalo plugin disable](#buffalo-plugin-disable)
- [Plugin Locations](#plugin-locations)
- [Configuration](#configuration)
- [Examples](#examples)
- [Best Practices](#best-practices)

---

## Overview

Buffalo plugin CLI allows you to manage plugins from the command line. Plugins can be:
- **Built-in**: Compiled into Buffalo (e.g., `naming-validator`)
- **External**: Loaded from `.so` files (Linux/macOS) or `.dll` (Windows)
- **Local**: Project-specific plugins in `./plugins/`
- **Global**: System-wide plugins in `~/.buffalo/plugins/`

---

## Commands

### buffalo plugin list

Lists all available plugins (built-in, local, and global).

**Usage:**
```bash
buffalo plugin list [flags]
```

**Flags:**
- (none)

**Example:**
```bash
$ buffalo plugin list

Built-in plugins:
  ✓ naming-validator (built-in)

Global plugins (~/.buffalo/plugins/):
  ✓ custom-linter
  ✓ doc-generator

Local plugins (./plugins/):
  ✓ project-validator

Plugins in configuration:
  naming-validator (enabled)
  custom-linter (disabled)
```

**Output Format:**
- `✓` - Plugin found and loadable
- `(built-in)` - Compiled into Buffalo
- `(enabled)` / `(disabled)` - Status in buffalo.yaml

---

### buffalo plugin install

Installs a plugin from a URL or local file.

**Usage:**
```bash
buffalo plugin install [source] --name [plugin-name] [flags]
```

**Arguments:**
- `source` - URL or local path to plugin `.so` file

**Required Flags:**
- `--name, -n` - Plugin name (used for configuration and file naming)

**Optional Flags:**
- `--global, -g` - Install globally to `~/.buffalo/plugins/` (default: local `./plugins/`)
- `--force, -f` - Overwrite if plugin already exists

**Examples:**

1. **Install from URL (local)**:
   ```bash
   buffalo plugin install https://example.com/plugins/v1.0.0/myplugin.so \
     --name myplugin
   ```

2. **Install from local file**:
   ```bash
   buffalo plugin install ./build/myplugin.so --name myplugin
   ```

3. **Install globally**:
   ```bash
   buffalo plugin install https://example.com/myplugin.so \
     --name myplugin --global
   ```

4. **Force overwrite**:
   ```bash
   buffalo plugin install ./myplugin.so --name myplugin --force
   ```

**Output:**
```
✓ Plugin 'myplugin' installed to ./plugins/myplugin.so

To enable this plugin, add it to your buffalo.yaml:

plugins:
  - name: myplugin
    enabled: true
    hook_points: [pre-build, post-parse, post-build]
    priority: 100
```

**Error Handling:**
- If plugin exists and `--force` not used: Returns error
- If source not found: Returns error
- If download fails: Returns error with HTTP status

---

### buffalo plugin remove

Removes an installed plugin file.

**Usage:**
```bash
buffalo plugin remove [name] [flags]
```

**Aliases:**
- `rm`
- `delete`

**Arguments:**
- `name` - Plugin name (without `.so` extension)

**Optional Flags:**
- `--global, -g` - Remove from global `~/.buffalo/plugins/` (default: local `./plugins/`)

**Examples:**

1. **Remove local plugin**:
   ```bash
   buffalo plugin remove myplugin
   ```

2. **Remove global plugin**:
   ```bash
   buffalo plugin remove myplugin --global
   ```

**Output:**
```
✓ Plugin 'myplugin' removed from ./plugins/myplugin.so

Note: Plugin is still in buffalo.yaml config. Use 'buffalo plugin disable' to disable it.
```

**Notes:**
- Only removes the `.so` file
- Does NOT modify buffalo.yaml
- Use `buffalo plugin disable` to disable in config

---

### buffalo plugin enable

Enables a plugin in buffalo.yaml configuration.

**Usage:**
```bash
buffalo plugin enable [name]
```

**Arguments:**
- `name` - Plugin name to enable

**Behavior:**
1. If plugin exists in config → Sets `enabled: true`
2. If plugin not in config → Adds new entry with default settings

**Default Settings:**
```yaml
- name: plugin-name
  enabled: true
  hook_points: [pre-build, post-parse, post-build]
  priority: 100
  options: {}
```

**Example:**
```bash
$ buffalo plugin enable naming-validator

✓ Plugin 'naming-validator' enabled in ./buffalo.yaml
```

**Config File:**
The command modifies your `buffalo.yaml`:
```yaml
plugins:
  - name: naming-validator
    enabled: true
    hook_points: [pre-build, post-parse, post-build]
    priority: 100
    options:
      strict_mode: true
```

**Notes:**
- Creates buffalo.yaml if not exists
- Preserves existing plugin options
- Safe to run multiple times (idempotent)

---

### buffalo plugin disable

Disables a plugin in buffalo.yaml configuration.

**Usage:**
```bash
buffalo plugin disable [name]
```

**Arguments:**
- `name` - Plugin name to disable

**Behavior:**
- Sets `enabled: false` in buffalo.yaml
- Keeps plugin configuration (options, hook_points, priority)

**Example:**
```bash
$ buffalo plugin disable naming-validator

✓ Plugin 'naming-validator' disabled in ./buffalo.yaml
```

**Config File:**
```yaml
plugins:
  - name: naming-validator
    enabled: false  # Changed from true
    hook_points: [pre-build, post-parse, post-build]
    priority: 100
```

**Notes:**
- Plugin file remains installed
- Configuration preserved for re-enabling
- Build will skip disabled plugins

---

## Plugin Locations

Buffalo searches for plugins in these locations (in order):

1. **Built-in plugins** (highest priority)
   - Compiled into Buffalo binary
   - Always available
   - Example: `naming-validator`

2. **Local plugins**
   - Directory: `./plugins/`
   - Project-specific
   - Versioned with project

3. **Global plugins**
   - Directory: `~/.buffalo/plugins/`
   - User-wide installation
   - Shared across projects

**Priority:**
- If same plugin name exists in multiple locations, built-in > local > global

---

## Configuration

Plugins are configured in `buffalo.yaml`:

```yaml
plugins:
  - name: naming-validator
    enabled: true
    hook_points:
      - pre-build
      - post-parse
      - post-build
    priority: 100
    options:
      strict_mode: true
      allow_uppercase: false
      
  - name: custom-linter
    enabled: false
    hook_points:
      - post-parse
    priority: 50
    options:
      rules: ["all"]
```

### Configuration Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Plugin identifier |
| `enabled` | boolean | Yes | Whether to load plugin |
| `hook_points` | []string | Yes | When to execute plugin |
| `priority` | int | Yes | Execution order (higher = earlier) |
| `options` | map | No | Plugin-specific options |

### Hook Points

Available hook points:
- `pre-build` - Before build starts
- `post-parse` - After proto files parsed
- `pre-compile` - Before compilation
- `post-compile` - After compilation
- `post-build` - After build completes
- `on-error` - On build errors

### Priority

Plugins execute in priority order within each hook:
- Higher priority = executes first
- Default: 100
- Range: 0-1000 (recommended)

---

## Examples

### Example 1: Install and Enable Built-in Plugin

```bash
# List available plugins
buffalo plugin list

# Enable naming-validator
buffalo plugin enable naming-validator

# Verify in config
cat buffalo.yaml

# Test
buffalo build
```

### Example 2: Install Custom Plugin from URL

```bash
# Download and install
buffalo plugin install \
  https://github.com/user/repo/releases/download/v1.0.0/myplugin.so \
  --name myplugin

# Enable in config
buffalo plugin enable myplugin

# Configure options (edit buffalo.yaml)
nano buffalo.yaml
# Add options:
#   options:
#     setting1: value1

# Test
buffalo build
```

### Example 3: Develop and Install Local Plugin

```bash
# Build your plugin
cd my-plugin
go build -buildmode=plugin -o myplugin.so

# Install locally
buffalo plugin install ./myplugin.so --name myplugin

# Enable
buffalo plugin enable myplugin

# Test
buffalo build

# Debug
buffalo build --verbose
```

### Example 4: Manage Multiple Plugins

```bash
# Install multiple plugins
buffalo plugin install plugin1.so --name plugin1
buffalo plugin install plugin2.so --name plugin2 --global
buffalo plugin install plugin3.so --name plugin3

# Enable specific ones
buffalo plugin enable plugin1
buffalo plugin enable plugin3

# List status
buffalo plugin list

# Disable temporarily
buffalo plugin disable plugin1

# Remove unused
buffalo plugin remove plugin2 --global
```

### Example 5: CI/CD Integration

```yaml
# .github/workflows/buffalo.yml
- name: Install Buffalo Plugin
  run: |
    buffalo plugin install \
      https://cdn.example.com/plugins/ci-linter.so \
      --name ci-linter --global

- name: Enable Plugin
  run: buffalo plugin enable ci-linter

- name: Build with Plugin
  run: buffalo build
```

---

## Best Practices

### 1. Version Management

Pin plugin versions in documentation:
```bash
# Good: Versioned URL
buffalo plugin install \
  https://example.com/plugins/v1.2.3/myplugin.so \
  --name myplugin

# Avoid: Latest/unversioned URLs
buffalo plugin install \
  https://example.com/plugins/latest/myplugin.so \
  --name myplugin
```

### 2. Local vs Global

**Use Local (`./plugins/`) for:**
- Project-specific plugins
- Plugins that should be versioned
- Team collaboration

**Use Global (`~/.buffalo/plugins/`) for:**
- Personal development tools
- CI/CD plugins
- System-wide utilities

### 3. Configuration in Version Control

**Include in Git:**
- `buffalo.yaml` (with plugin configuration)
- Local plugins (`./plugins/*.so`)
- Plugin documentation

**Exclude from Git:**
- Global plugins (`~/.buffalo/plugins/`)
- Build artifacts

### 4. Plugin Discovery

Document plugins in project README:
```markdown
## Required Plugins

Install required plugins:
```bash
buffalo plugin install https://example.com/naming-validator.so --name naming-validator
buffalo plugin install https://example.com/doc-gen.so --name doc-gen
```

Or use initialization script: `./scripts/install-plugins.sh`
```

### 5. Testing

Test plugins before committing:
```bash
# Dry-run to check plugin behavior
buffalo build --dry-run

# Run with verbose logging
buffalo build --verbose

# Doctor check
buffalo doctor
```

### 6. Error Handling

Handle plugin errors gracefully:
```yaml
plugins:
  - name: optional-linter
    enabled: true
    hook_points: [post-parse]
    priority: 50
    options:
      fail_on_error: false  # Continue build on plugin errors
```

### 7. Documentation

Document plugin options:
```yaml
plugins:
  - name: naming-validator
    enabled: true
    hook_points: [pre-build]
    priority: 100
    options:
      # Enforce snake_case naming
      strict_mode: true
      # Allow uppercase in enum values
      allow_uppercase_enums: true
      # Custom naming patterns
      patterns:
        - "^[a-z][a-z0-9_]*$"
```

---

## Troubleshooting

### Plugin Not Found

**Error:**
```
Error: plugin myplugin not found
```

**Solutions:**
1. Check plugin installed:
   ```bash
   buffalo plugin list
   ls ./plugins/  # or ~/.buffalo/plugins/
   ```

2. Verify plugin name matches config:
   ```bash
   grep "name:" buffalo.yaml
   ```

3. Re-install:
   ```bash
   buffalo plugin install [source] --name myplugin --force
   ```

### Plugin Load Failed

**Error:**
```
Error: failed to load plugin: plugin.Open: ...
```

**Solutions:**
1. Check Go version compatibility (plugin must match Buffalo's Go version)
2. Verify plugin architecture (amd64, arm64, etc.)
3. Check plugin dependencies
4. Rebuild plugin with correct flags:
   ```bash
   go build -buildmode=plugin -o myplugin.so
   ```

### Plugin Execution Failed

**Error:**
```
Error: plugin naming-validator validation failed with 4 error(s)
```

**Solutions:**
1. Review plugin output (fix proto files):
   ```bash
   buffalo build --verbose
   ```

2. Disable plugin temporarily:
   ```bash
   buffalo plugin disable naming-validator
   buffalo build
   ```

3. Adjust plugin options in buffalo.yaml:
   ```yaml
   options:
     strict_mode: false
   ```

### Permission Denied

**Error:**
```
Error: permission denied: /usr/local/bin/buffalo
```

**Solution:**
```bash
# Use sudo for global install
sudo buffalo plugin install [source] --name myplugin --global

# Or install locally (no sudo needed)
buffalo plugin install [source] --name myplugin
```

---

## See Also

- [Plugin Development Guide](./PLUGIN_GUIDE.md)
- [Plugin API Reference](./PLUGIN_API.md)
- [CLI Commands Reference](./CLI_COMMANDS.md)
- [Configuration Guide](./CONFIGURATION.md)

---

**Version:** 0.7.0  
**Last Updated:** January 2026
