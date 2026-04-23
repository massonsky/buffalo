"""Module extensions for rules_buffalo (bzlmod)."""

load(
    ":repositories.bzl",
    "DEFAULT_BUFFALO_REPO",
    "DEFAULT_BUFFALO_VERSION",
    "DEFAULT_GRPCIO_TOOLS_VERSION",
    "DEFAULT_PROTOBUF_PY_VERSION",
    "DEFAULT_PROTOC_GEN_GO_GRPC_VERSION",
    "DEFAULT_PROTOC_GEN_GO_VERSION",
    "DEFAULT_PROTOC_VERSION",
    "buffalo_toolchain_repo",
)

_TOOLCHAIN_TAG = tag_class(
    attrs = {
        "buffalo_version": attr.string(default = DEFAULT_BUFFALO_VERSION),
        "buffalo_repo": attr.string(default = DEFAULT_BUFFALO_REPO),
        "protoc_version": attr.string(default = DEFAULT_PROTOC_VERSION),
        "protoc_gen_go_version": attr.string(default = DEFAULT_PROTOC_GEN_GO_VERSION),
        "protoc_gen_go_grpc_version": attr.string(default = DEFAULT_PROTOC_GEN_GO_GRPC_VERSION),
        "grpcio_tools_version": attr.string(default = DEFAULT_GRPCIO_TOOLS_VERSION),
        "protobuf_version": attr.string(default = DEFAULT_PROTOBUF_PY_VERSION),
        "integrity": attr.string_dict(
            default = {},
            doc = "Optional sha256 integrity overrides; see buffalo_toolchain_repo for the key format.",
        ),
    },
    doc = "Configures pinned tool versions for the Buffalo toolchain.",
)

def _select_config(module_ctx):
    cfg = struct(
        buffalo_version = DEFAULT_BUFFALO_VERSION,
        buffalo_repo = DEFAULT_BUFFALO_REPO,
        protoc_version = DEFAULT_PROTOC_VERSION,
        protoc_gen_go_version = DEFAULT_PROTOC_GEN_GO_VERSION,
        protoc_gen_go_grpc_version = DEFAULT_PROTOC_GEN_GO_GRPC_VERSION,
        grpcio_tools_version = DEFAULT_GRPCIO_TOOLS_VERSION,
        protobuf_version = DEFAULT_PROTOBUF_PY_VERSION,
        integrity = {},
    )

    # Last-wins: the root module's tag overrides any tags from transitive deps.
    seen_root = False
    for mod in module_ctx.modules:
        for tag in mod.tags.toolchain:
            if mod.is_root or not seen_root:
                cfg = struct(
                    buffalo_version = tag.buffalo_version,
                    buffalo_repo = tag.buffalo_repo,
                    protoc_version = tag.protoc_version,
                    protoc_gen_go_version = tag.protoc_gen_go_version,
                    protoc_gen_go_grpc_version = tag.protoc_gen_go_grpc_version,
                    grpcio_tools_version = tag.grpcio_tools_version,
                    protobuf_version = tag.protobuf_version,
                    integrity = dict(tag.integrity),
                )
                if mod.is_root:
                    seen_root = True
    return cfg

def _buffalo_impl(module_ctx):
    cfg = _select_config(module_ctx)
    buffalo_toolchain_repo(
        name = "buffalo_toolchain",
        buffalo_version = cfg.buffalo_version,
        buffalo_repo = cfg.buffalo_repo,
        protoc_version = cfg.protoc_version,
        protoc_gen_go_version = cfg.protoc_gen_go_version,
        protoc_gen_go_grpc_version = cfg.protoc_gen_go_grpc_version,
        grpcio_tools_version = cfg.grpcio_tools_version,
        protobuf_version = cfg.protobuf_version,
        integrity = cfg.integrity,
        python_interpreter = Label("@python_3_12_host//:python"),
    )

buffalo = module_extension(
    implementation = _buffalo_impl,
    tag_classes = {"toolchain": _TOOLCHAIN_TAG},
    doc = "Provisions Buffalo CLI + protoc + Go/Python plugins as a hermetic toolchain.",
)
