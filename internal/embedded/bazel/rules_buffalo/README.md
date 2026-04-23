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

`rules_buffalo` is **hermetic by default**: Bazel itself downloads pinned,
prebuilt binaries of `protoc`, the Go plugins, and the Buffalo CLI from
upstream releases. **No host `go`, no host `protoc`, no host plugins are
required.**

What is downloaded automatically on first build:

| Tool | Source |
|------|--------|
| `protoc` | `protocolbuffers/protobuf` GitHub Releases |
| `protoc-gen-go` | `protocolbuffers/protobuf-go` GitHub Releases |
| `protoc-gen-go-grpc` | `grpc/grpc-go` GitHub Releases |
| `buffalo` CLI | `massonsky/buffalo` GitHub Releases (configurable) |

Supported host platforms: linux/amd64, linux/arm64, darwin/amd64,
darwin/arm64, windows/amd64 (windows/arm64 falls back to amd64).

### Plugins not yet hermetic

Until their respective integrations are wired up, the following still fall
back to host tools when used:

- `protoc-gen-grpc_python` — bootstrapped from host `python` + `pip`
  (`grpcio-tools`). Will move to `rules_python` in a follow-up.
- `protoc-gen-prost` / `protoc-gen-tonic` — bootstrapped from host `cargo`.
  Will move to `rules_rust` in a follow-up.

If you don't generate Python or Rust code, you don't need Python or Rust on
your host.

### Pinning versions

Defaults can be overridden via environment variables (set them in `.bazelrc`
with `common --repo_env=NAME=value`):

| Variable | Default |
|----------|---------|
| `BUFFALO_TOOLCHAIN_PROTOC_VERSION` | `25.1` |
| `BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_VERSION` | `1.34.2` |
| `BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_GRPC_VERSION` | `1.5.1` |
| `BUFFALO_TOOLCHAIN_BUFFALO_VERSION` | `4.0.0` |
| `BUFFALO_TOOLCHAIN_BUFFALO_REPO` | `massonsky/buffalo` |

### Pinning to internal mirrors / arbitrary URLs

Set a direct URL for any tool to skip the auto-picked upstream URL
(useful for air-gapped CI):

- `BUFFALO_TOOLCHAIN_BUFFALO_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_GRPC_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_GEN_GRPC_PYTHON_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_GEN_PROST_URL`
- `BUFFALO_TOOLCHAIN_PROTOC_GEN_TONIC_URL`

### Strict mode

`BUFFALO_TOOLCHAIN_STRICT_SANDBOX=1` disables every host fallback. In this
mode `protoc-gen-grpc_python` (and Rust plugins, if used) **must** be
provided via the URL env vars above; otherwise the build fails.

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

- `grpc_tools.protoc` — Protocol Buffer + gRPC code generation. Currently
  bootstrapped from host `python` + `pip` (will become hermetic via
  `rules_python`).

### Rust

- `protoc-gen-prost` — Protocol Buffer code generation (default, via host
  `cargo install`; will become hermetic via `rules_rust`).
- `protoc-gen-tonic` — gRPC support with Tokio runtime (optional, via host
  `cargo install`).

### TypeScript

- Requires `npm` and Node.js toolchain (installed separately)

### C++

- Requires C++ compiler (`clang++`, `g++`, or MSVC) in PATH
- Uses protobuf C++ runtime library
