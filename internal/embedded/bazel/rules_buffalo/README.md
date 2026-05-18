# rules_buffalo

Bazel rules for [Buffalo](https://github.com/massonsky/buffalo) — multi-language protobuf/gRPC code generator.

**Fully hermetic.** Bazel itself downloads `protoc`, the Go plugins, the Python
gRPC plugin (with its own hermetic Python interpreter), the Rust prost/tonic
generators, TypeScript support when enabled, and the Buffalo CLI. **No host
`go`, no host `python`, no host `protoc`, no host plugins are required.**

## Setup (bzlmod)

`rules_buffalo` is not yet published to the Bazel Central Registry. Use
`archive_override` to pin a release by version tag (recommended — no
commit SHA needed):

```python
# MODULE.bazel
bazel_dep(name = "rules_buffalo", version = "1.0.0")
archive_override(
    module_name = "rules_buffalo",
    urls = ["https://github.com/massonsky/buffalo/archive/refs/tags/v{{BUFFALO_VERSION}}.tar.gz"],
    strip_prefix = "buffalo-{{BUFFALO_VERSION}}/bazel/rules_buffalo",
    # integrity = "sha256-...",  # optional but recommended for production
)

buffalo = use_extension("@rules_buffalo//buffalo:extensions.bzl", "buffalo")
use_repo(buffalo, "buffalo_toolchain")
```

If you need a specific commit instead of a release tag, use `git_override`:

```python
git_override(
    module_name = "rules_buffalo",
    remote = "https://github.com/massonsky/buffalo.git",
    commit = "<sha>",
    strip_prefix = "bazel/rules_buffalo",
)
```

For local development of `rules_buffalo` itself:

```python
local_path_override(
    module_name = "rules_buffalo",
    path = "path/to/rules_buffalo",
)
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
    buffalo_version            = "{{BUFFALO_VERSION}}",
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

### Pinning sha256 integrity (production)

For reproducible, tamper-proof builds, lock the sha256 of every downloaded
artifact via the `integrity` attribute. On first build Bazel prints the
expected integrity hash for each artifact; copy them into your
`MODULE.bazel`:

```python
buffalo.toolchain(
    integrity = {
        "protoc-25.1-win64":                   "sha256-...",
        "protoc-25.1-linux-x86_64":            "sha256-...",
        "protoc-gen-go-1.34.2-linux-amd64":    "sha256-...",
        "protoc-gen-go-grpc-1.5.1-linux-amd64": "sha256-...",
        "buffalo-{{BUFFALO_VERSION}}-linux-amd64": "sha256-...",
    },
)
```

Key format:
- `protoc-<v>-<plat>` (plat = `linux-x86_64`, `linux-aarch_64`, `osx-x86_64`,
  `osx-aarch_64`, `win64`)
- `protoc-gen-go-<v>-<os>-<arch>`
- `protoc-gen-go-grpc-<v>-<os>-<arch>`
- `buffalo-<v>-<os>-<arch>`

Without integrity entries, downloads still work and are pinned by URL
(HTTPS + immutable GitHub releases), but they are not cryptographically
verified.

## Rules

### `buffalo_proto_compile`

Compiles proto files using Buffalo within the Bazel sandbox.
Output is a tree artifact in `bazel-out/`. By default the artifact is named
`generated`, matching `buffalo init`'s `output.base_dir: ./generated`.

```python
load("@rules_buffalo//buffalo:defs.bzl", "buffalo_proto_compile")

buffalo_proto_compile(
    name = "proto_gen",
    srcs = glob(["proto/**/*.proto"]),
    config = "buffalo.yaml",
    languages = ["go", "python"],
    out = "generated",  # keep aligned with output.base_dir in buffalo.yaml
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
Generated code goes into the source tree path configured by
`output.base_dir` (e.g., `generated/`).

```python
load("@rules_buffalo//buffalo:defs.bzl", "buffalo_proto_gen")

buffalo_proto_gen(
    name = "buffalo_gen",
    srcs = glob(["proto/**/*.proto"]),
    config = "buffalo.yaml",
    languages = ["go", "python"],
    out = "generated",  # keep aligned with output.base_dir in buffalo.yaml
)
```

Usage:

```bash
bazel run //:buffalo_gen
bazel run //:buffalo_gen -- --verbose
```

With `srcs`, `buffalo_proto_gen` creates a private
`<name>_compile` target that uses the same hermetic toolchain as
`buffalo_proto_compile`. `bazel run` builds that target, then copies the
compiled tree artifact from `bazel-bin` into the source-tree directory from
`output.base_dir`.

## Attributes

- `srcs` (`label_list`, optional): proto source files; enables hermetic compile-and-copy mode for `bazel run`
- `config` (`label`, default: `None`): Buffalo config file (`buffalo.yaml`)
- `languages` (`string_list`, default: `["go", "python"]`): target languages
- `proto_paths` (`string_list`, default: `["proto"]`): proto import directories
- `deps` (`label_list`, default: `[]`): additional proto dependencies
- `out` (`string`, default: `"generated"`): Bazel tree artifact name; keep it aligned with `output.base_dir` in `buffalo.yaml`
- `compile_target` (`label`, optional): existing `buffalo_proto_compile` target to copy from
- `compile_out` (`string`, optional): path under `bazel-bin`; defaults to the package path plus `out`
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
| Rust | ✅ Hermetic (opt-in) | `protoc-gen-prost` + `protoc-gen-tonic` |
| TypeScript | ✅ Hermetic (opt-in) | `ts-proto` via hermetic Node.js |

### Enabling Rust

Rust is opt-in to keep download size minimal for projects that don't use it.
`buffalo.rust()` enables the hermetic `protoc-gen-prost` and
`protoc-gen-tonic` binaries bundled by `rules_buffalo` through its internal
`rules_rust` / `crate_universe` setup. Consuming projects do not need to declare
`rules_rust`, `crate_universe`, or a separate `buffalo_rust_plugins` repo.

```python
# MODULE.bazel
buffalo = use_extension("@rules_buffalo//buffalo:extensions.bzl", "buffalo")
buffalo.rust(
    protoc_gen_prost_version = "0.4.0",
    protoc_gen_tonic_version = "0.4.1",
)
use_repo(buffalo, "buffalo_toolchain")
```

Then `buffalo_proto_compile` can use Rust without per-target plugin labels:

```python
load("@rules_buffalo//buffalo:defs.bzl", "buffalo_proto_compile")

buffalo_proto_compile(
    name = "proto_gen",
    srcs = glob(["proto/**/*.proto"]),
    config = "buffalo.yaml",
    languages = ["rust"],
)
```

Without `buffalo.rust()`, `rules_buffalo` provides stub targets so non-Rust
builds work without forcing every project to download and build Rust plugin
crates. To use custom Rust generator versions, provide your own binary labels
via `buffalo.rust(protoc_gen_prost = ..., protoc_gen_tonic = ...)`.

### Enabling TypeScript

TypeScript is opt-in. Add a `buffalo.typescript()` tag to enable the
hermetic [`ts-proto`](https://github.com/stephenh/ts-proto) plugin. A
prebuilt Node.js runtime is downloaded from `nodejs.org` and `ts-proto` is
installed into a private `node_modules` directory inside the toolchain repo:

```python
buffalo = use_extension("@rules_buffalo//buffalo:extensions.bzl", "buffalo")
buffalo.typescript()  # enables ts-proto with default Node + ts-proto versions
use_repo(buffalo, "buffalo_toolchain")
```

Optional version pinning:

```python
buffalo.typescript(
    node_version = "20.18.0",
    ts_proto_version = "1.181.2",
)
```

Supported platforms: linux/darwin (x64+arm64) and windows (x64). No global
Node.js installation is required on the host.
