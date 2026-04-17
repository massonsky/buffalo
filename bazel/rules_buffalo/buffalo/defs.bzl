"""Buffalo proto compilation rules for Bazel.

Provides two integration modes:

1. buffalo_proto_compile — hermetic Bazel rule
   Compiles proto files within the Bazel sandbox.
   Output is a tree artifact in bazel-out/.
   Downstream targets depend on this rule's output.

2. buffalo_proto_gen — source-tree generation macro
   Creates a `bazel run` target that invokes Buffalo in the workspace root.
   Generated code goes into the source tree (gen/).
   Use this for development workflows.
"""

# ── Providers ─────────────────────────────────────────────────────────────────

BuffaloProtoInfo = provider(
    doc = "Information about Buffalo proto compilation output.",
    fields = {
        "generated_dir": "Tree artifact with generated source files.",
        "languages": "List of languages that were generated.",
        "proto_srcs": "Original proto source files.",
    },
)

# ── buffalo_proto_compile (hermetic rule) ─────────────────────────────────────

def _buffalo_proto_compile_impl(ctx):
    output_dir = ctx.actions.declare_directory(ctx.attr.out)

    # Build command parts — shell-safe, one per element
    cmd_parts = [
        "buffalo", "build",
        "--skip-system-check",
        "--skip-lock",
    ]

    if ctx.file.config:
        cmd_parts.extend(["--config", ctx.file.config.path])

    cmd_parts.extend(["-o", output_dir.path])

    for lang in ctx.attr.languages:
        cmd_parts.extend(["-l", lang])

    for p in ctx.attr.proto_paths:
        cmd_parts.extend(["-p", p])

    if ctx.attr.verbose:
        cmd_parts.append("--verbose")

    # Collect all inputs
    inputs = list(ctx.files.srcs)
    if ctx.file.config:
        inputs.append(ctx.file.config)
    inputs.extend(ctx.files.deps)

    ctx.actions.run_shell(
        outputs = [output_dir],
        inputs = inputs,
        command = " ".join(cmd_parts),
        mnemonic = "BuffaloCompile",
        progress_message = "Buffalo: compiling %d proto files [%s]" % (
            len(ctx.files.srcs),
            ", ".join(ctx.attr.languages),
        ),
        use_default_shell_env = True,
    )

    return [
        DefaultInfo(files = depset([output_dir])),
        BuffaloProtoInfo(
            generated_dir = output_dir,
            languages = ctx.attr.languages,
            proto_srcs = ctx.files.srcs,
        ),
    ]

buffalo_proto_compile = rule(
    implementation = _buffalo_proto_compile_impl,
    attrs = {
        "srcs": attr.label_list(
            allow_files = [".proto"],
            mandatory = True,
            doc = "Proto source files to compile.",
        ),
        "config": attr.label(
            allow_single_file = [".yaml", ".yml"],
            doc = "Buffalo configuration file (buffalo.yaml).",
        ),
        "languages": attr.string_list(
            default = ["go", "python", "rust"],
            doc = "Target languages for code generation.",
        ),
        "proto_paths": attr.string_list(
            default = ["proto"],
            doc = "Directories to search for proto imports.",
        ),
        "deps": attr.label_list(
            allow_files = True,
            doc = "Additional proto file dependencies (e.g., third-party protos).",
        ),
        "out": attr.string(
            default = "gen",
            doc = "Output directory name (becomes a tree artifact).",
        ),
        "verbose": attr.bool(
            default = False,
            doc = "Enable verbose Buffalo output.",
        ),
    },
    doc = """Compiles proto files using Buffalo.

Produces a tree artifact containing generated source code for the
specified languages. The output directory structure mirrors what
`buffalo build` produces: `<out>/<lang>/...`.

Example:
    buffalo_proto_compile(
        name = "proto_gen",
        srcs = glob(["proto/**/*.proto"]),
        config = "buffalo.yaml",
        languages = ["go", "rust", "python"],
    )
""",
)

# ── buffalo_proto_gen (source-tree run target) ────────────────────────────────

def buffalo_proto_gen(
        name,
        config = "buffalo.yaml",
        languages = [],
        proto_paths = [],
        extra_args = [],
        visibility = None,
        **kwargs):
    """Creates a `bazel run` target that invokes Buffalo in the workspace root.

    Generated code goes into the source tree (e.g., gen/).
    Useful for development workflows where downstream targets
    reference files in the source tree.

    Usage:
        bazel run //:buffalo_gen
        bazel run //:buffalo_gen -- --verbose

    Args:
        name: Target name.
        config: Path to buffalo.yaml (relative to workspace root).
        languages: List of target languages (e.g., ["go", "rust"]).
        proto_paths: Directories to search for proto imports.
        extra_args: Additional CLI arguments for `buffalo build`.
        visibility: Bazel visibility.
    """
    args = ["--config", config]
    for lang in languages:
        args.extend(["-l", lang])
    for p in proto_paths:
        args.extend(["-p", p])
    args.extend(extra_args)

    native.py_binary(
        name = name,
        srcs = [Label("//buffalo:gen_runner.py")],
        main = Label("//buffalo:gen_runner.py"),
        args = args,
        visibility = visibility,
        **kwargs
    )
