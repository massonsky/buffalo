# rules_buffalo

Bazel rules for [Buffalo](https://github.com/massonsky/buffalo) — multi-language protobuf/gRPC code generator.

**Fully hermetic.** Bazel itself downloads everything: `protoc`, the Go
plugins, the Python gRPC plugin (with its own hermetic Python interpreter),
and the Buffalo CLI. **No host `go`, no host `python`, no host `protoc`,
no host plugins are required.**

## Setup (bzlmod)

In your `MODULE.bazel`:

```python
bazel_dep(name = "rules_buffalo", version = "1.0.0")

# Optional — only needed for local development of rules_buffalo itself.
local_path_override(
    module_name = "rules_buffalo",
    path = "path/to/rules_buffalo",
)

buffalo = use_extension("@rules_buffalo//buffalo:extensions.bzl", "buffalo")
use_repo(buffalo, "buffalo_toolchain")
```

That's it. On first build Bazel downloads:

| Tool | Source |
|------|--------|
| `protoc` | `protocolbuffers/protobuf` GitHub Releases |
| `protoc-gen-go` | `protocolbuffers/protobuf-go` GitHub Releases |
| `protoc-gen-go-grpc` | `grpc/grpc-go` GitHub Releases |
| `protoc-gen-grpc_python` | `grpcio-tools` wheel into hermetic Python from `rules_python` |
| `buffalo` CLI | `massonsky/buffalo` GitHub Releases |

Supported host platforms: linux/amd64, linux/arm64, darwin/amd64,
darwin/arm64, windows/amd64 (windows/arm64 falls back to amd64).

### Pinning versions (optional)

Defaults are good. To pin a different version, add a `toolchain` tag:

```python
buffalo = use_extension("@rules_buffalo//buffalo:extensions.bzl", "buffalo")
buffalo.toolchain(
    buffalo_version            = "4.1.0",
    protoc_version             = "28.2",
    protoc_gen_go_version      = "1.34.2",
    protoc_gen_go_grpc_version = "1.5.1",
    grpcio_tools_version       = "1.64.1",
    protobuf_version           = "5.27.1",
)
use_repo(buffalo, "buffalo_toolchain")
```

All attributes are optional; omitted ones use the defaults baked into
`rules_buffalo`.

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
    languages = ["go", "python"],
)
```

Per-language targets for fine-grained dependencies:

```python
buffalo_proto_compile(
    name = "proto_gen_python",
    srcs = glob(["proto/**/*.proto"]),
    config = "buffalo.yaml",
    languages = ["python"],
    out = "proto_gen_python",
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
    languages = ["go", "python"],
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
- `languages` (`string_list`, default: `["go", "python"]`): target languages
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

| Language | Status | Plugin |
|----------|--------|--------|
| Go | ✅ Hermetic | `protoc-gen-go` + `protoc-gen-go-grpc` |
| Python | ✅ Hermetic | `grpc_tools.protoc` via hermetic CPython |
| C++ | ✅ Hermetic | built into `protoc` |
| Rust | 🚧 Planned | `protoc-gen-prost` / `protoc-gen-tonic` via `rules_rust` |
| TypeScript | 🚧 Planned | via `rules_nodejs` |

Rust and TypeScript will be added in follow-up commits with their respective
`rules_rust` and `rules_nodejs` integrations. Until then they are not
available in the hermetic toolchain.
