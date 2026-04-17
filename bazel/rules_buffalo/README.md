# rules_buffalo

Bazel rules for [Buffalo](https://github.com/massonsky/buffalo) — multi-language protobuf/gRPC code generator.

## Setup (bzlmod)

In your `MODULE.bazel`:

```python
bazel_dep(name = "rules_buffalo", version = "1.0.0")

# For local development:
local_path_override(
    module_name = "rules_buffalo",
    path = "path/to/rules_buffalo",
)

# Register buffalo toolchain (finds binary in PATH)
buffalo = use_extension("@rules_buffalo//buffalo:extensions.bzl", "buffalo")
use_repo(buffalo, "buffalo_toolchain")
```

## Prerequisites

Buffalo must be installed and available in `PATH`:

```bash
go install github.com/massonsky/buffalo/cmd/buffalo@latest
```

## Rules

### `buffalo_proto_compile`

Compiles proto files using Buffalo within the Bazel sandbox.
Output is a tree artifact in `bazel-out/`.

```python
load("@rules_buffalo//buffalo:defs.bzl", "buffalo_proto_compile")

buffalo_proto_compile(
    name = "proto_gen",
    srcs = glob(["proto/**/*.proto"]),
    config = "buffalo.yaml",
    languages = ["go", "rust", "python"],
)
```

Per-language targets for fine-grained dependencies:

```python
buffalo_proto_compile(
    name = "proto_gen_rust",
    srcs = glob(["proto/**/*.proto"]),
    config = "buffalo.yaml",
    languages = ["rust"],
    out = "proto_gen_rust",
)
```

### `buffalo_proto_gen`

Macro that creates a `bazel run` target for source-tree generation.
Generated code goes into the source tree (e.g., `gen/`).

```python
load("@rules_buffalo//buffalo:defs.bzl", "buffalo_proto_gen")

buffalo_proto_gen(
    name = "buffalo_gen",
    config = "buffalo.yaml",
    languages = ["go", "rust", "python"],
)
```

Usage:

```bash
bazel run //:buffalo_gen
bazel run //:buffalo_gen -- --verbose
```

## Attributes

| Attribute | Type | Default | Description |
|-----------|------|---------|-------------|
| `srcs` | `label_list` | required | Proto source files |
| `config` | `label` | `None` | Buffalo config file (buffalo.yaml) |
| `languages` | `string_list` | `["go", "python", "rust"]` | Target languages |
| `proto_paths` | `string_list` | `["proto"]` | Proto import directories |
| `deps` | `label_list` | `[]` | Additional proto dependencies |
| `out` | `string` | `"gen"` | Output directory name |
| `verbose` | `bool` | `False` | Enable verbose output |

## Providers

### `BuffaloProtoInfo`

Returned by `buffalo_proto_compile`:

- `generated_dir` — Tree artifact with generated sources
- `languages` — Languages that were generated
- `proto_srcs` — Original proto source files
