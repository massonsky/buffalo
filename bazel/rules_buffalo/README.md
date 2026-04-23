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

`rules_buffalo` bootstraps required runtime tools into the Bazel external
toolchain repository used by sandboxed actions.

Important: runtime tools used by `buffalo_proto_compile` are staged in the
Bazel toolchain repository and executed from there, ensuring deterministic,
hermetic builds regardless of host environment.

**Default mode (compatibility):**

Requires host tools for initial bootstrap only:

- `go` in `PATH` (for auto-install of Go-based tools)
- `python`/`python3`/`py` in `PATH` (for grpc tools bootstrap)
- `cargo` in `PATH` (for auto-install of Rust tools)
- `rustc` in `PATH` (for Rust code generation)
- `npm` in `PATH` (for TypeScript support, optional)
- C++ compiler (`clang++`, `g++`, or MSVC `cl.exe` on Windows) for C++ support (optional)

First `bazel build` automatically installs:

- `buffalo` (via `go install` into toolchain repo)
- `protoc-gen-go` (via `go install` into toolchain repo)
- `protoc-gen-go-grpc` (via `go install` into toolchain repo)
- `protoc-gen-prost` (via `cargo install` into toolchain repo, for Rust)
- `protoc-gen-tonic` (via `cargo install` into toolchain repo, for Rust gRPC, optional)
- Python `grpcio-tools` / `protobuf` for `grpc_tools.protoc` (via `pip install --target` into toolchain repo)

**Strict mode** (`BUFFALO_TOOLCHAIN_STRICT_SANDBOX=1`):

No host tool dependencies. All tools are downloaded from URLs specified in environment variables.

## Strict hermetic sandbox mode

By default, compatibility mode is enabled (automatic bootstrap via `go`/`python`).
Bootstrap tooling is installed into the Bazel toolchain repository, allowing
reproducible sandbox execution.

For strict hermetic sandbox mode, set:

- `BUFFALO_TOOLCHAIN_STRICT_SANDBOX=1`

In strict mode, all runtime tooling must be provided via download URLs and is downloaded into the
toolchain repository (sandbox input), with no runtime dependency on host-installed
Buffalo/protoc/plugins or host `go`/`python`.

Required environment variables in strict mode:

- `BUFFALO_TOOLCHAIN_BUFFALO_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_GRPC_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_GEN_GRPC_PYTHON_URL`

### Compatibility mode (default)

By default, `BUFFALO_TOOLCHAIN_STRICT_SANDBOX` is disabled. In this mode:

1. Tools are auto-installed from source on first Bazel run
2. `protoc` is bootstrapped via `pip install --target grpcio-tools`
3. Go-based tools (`buffalo`, `protoc-gen-go`, `protoc-gen-go-grpc`) are installed via `go install`
4. Requires: `go` in PATH, `python`/`python3`/`py` in PATH
5. Generated actions still run with tools staged in Bazel sandbox (hermetic execution)

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

## Supported Languages

### Go

- `protoc-gen-go` - Protocol Buffer code generation
- `protoc-gen-go-grpc` - gRPC support via google.golang.org/grpc

### Python

- `grpc_tools.protoc` - Protocol Buffer and gRPC code generation (bootstrapped via pip)

### Rust

- `protoc-gen-prost` - Protocol Buffer code generation (default, via cargo install)
- `protoc-gen-tonic` - gRPC support with Tokio runtime (optional, via cargo install)

### TypeScript

- Requires `npm` and Node.js toolchain (installed separately)

### C++

- Requires C++ compiler (`clang++`, `g++`, or MSVC) in PATH
- Uses protobuf C++ runtime library
