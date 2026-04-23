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

`rules_buffalo` now bootstraps required runtime tools into the Bazel external
toolchain repository used by sandboxed actions.

Important: runtime tools used by `buffalo_proto_compile` are **not** taken
directly from host PATH. They are installed into (and executed from) the
toolchain repository.

Bootstrapped dependencies:

- `buffalo` (via `go install` into toolchain repo)
- `protoc-gen-go` (via `go install` into toolchain repo)
- `protoc-gen-go-grpc` (via `go install` into toolchain repo)
- Python `grpcio-tools` / `protobuf` for `grpc_tools.protoc` (via `pip install --target` into toolchain repo)

If bootstrap is not possible, setup fails with an actionable error message.

Minimal host prerequisites:

- `go` in `PATH` (for auto-install of Go-based tools)
- `python`/`python3`/`py` in `PATH` (for grpc tools bootstrap)

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

- `srcs` (`label_list`, required): proto source files
- `config` (`label`, default: `None`): Buffalo config file (`buffalo.yaml`)
- `languages` (`string_list`, default: `["go", "python", "rust"]`): target languages
- `proto_paths` (`string_list`, default: `["proto"]`): proto import directories
- `deps` (`label_list`, default: `[]`): additional proto dependencies
- `out` (`string`, default: `"gen"`): output directory name
- `verbose` (`bool`, default: `False`): enable verbose output

## Providers

### `BuffaloProtoInfo`

Returned by `buffalo_proto_compile`:

- `generated_dir` — Tree artifact with generated sources
- `languages` — Languages that were generated
- `proto_srcs` — Original proto source files
