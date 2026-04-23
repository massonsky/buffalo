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

load("@rules_python//python:defs.bzl", "py_binary")

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
    is_windows = ctx.configuration.host_path_separator == ";"
    helper_rel = ctx.file._config_output_reader.short_path
    staged_inputs = []
    if ctx.file.config:
        staged_inputs.append(ctx.file.config)
    staged_inputs.extend(ctx.files.srcs)
    staged_inputs.extend(ctx.files.deps)

    tool_files = [
        (ctx.file.buffalo, ctx.file.buffalo.basename),
        (ctx.file.protoc, ctx.file.protoc.basename),
        (ctx.file.protoc_gen_go, ctx.file.protoc_gen_go.basename),
        (ctx.file.protoc_gen_go_grpc, ctx.file.protoc_gen_go_grpc.basename),
    ]
    if ctx.file.protoc_gen_grpc_python:
        tool_files.append((ctx.file.protoc_gen_grpc_python, ctx.file.protoc_gen_grpc_python.basename))
    if ctx.file.protoc_gen_prost:
        tool_files.append((ctx.file.protoc_gen_prost, ctx.file.protoc_gen_prost.basename))
    if ctx.file.protoc_gen_tonic:
        tool_files.append((ctx.file.protoc_gen_tonic, ctx.file.protoc_gen_tonic.basename))

    command_parts = [
        "buffalo",
        "build",
        "--skip-system-check",
        "--skip-lock",
    ]

    if ctx.file.config:
        command_parts.extend(["--config", ctx.file.config.short_path])

    if not (ctx.attr.respect_config_output and ctx.file.config):
        command_parts.extend(["-o", output_dir.path])

    for lang in ctx.attr.languages:
        command_parts.extend(["-l", lang])

    for p in ctx.attr.proto_paths:
        command_parts.extend(["-p", p])

    for p in ctx.attr.import_paths:
        command_parts.extend(["-I", p])

    if ctx.attr.verbose:
        command_parts.append("--verbose")

    wrapper_ext = ".bat" if is_windows else ".sh"
    wrapper = ctx.actions.declare_file(ctx.label.name + "_buffalo" + wrapper_ext)

    if is_windows:
        normalized = [part.replace("/", "\\") for part in command_parts]
        quoted = []
        for index, part in enumerate(normalized):
            needs_quote = (
                index == 0 or
                part.startswith(".") or
                "\\" in part or
                "/" in part or
                ":" in part or
                " " in part
            )
            quoted.append("\"{}\"".format(part) if needs_quote else part)
        for index, part in enumerate(normalized):
            if part == "-o" and index + 1 < len(normalized):
                out_part = normalized[index + 1]
                if not (":" in out_part or out_part.startswith("\\")):
                    quoted[index + 1] = "\"%EXECROOT%\\{}\"".format(out_part)
        stage_setup = [
            "@echo off\r\n",
            "setlocal\r\n",
            "set \"EXECROOT=%CD%\"\r\n",
            "set \"CONFIG_OUTPUT={}\"\r\n".format(ctx.attr.out.replace("/", "\\")),
            "set \"STAGE=%TEMP%\\buffalo_{}_%RANDOM%\"\r\n".format(ctx.label.name),
            "set \"TOOLS=%STAGE%\\_buffalo_tools\"\r\n",
            "if exist \"%STAGE%\" rmdir /s /q \"%STAGE%\"\r\n",
            "mkdir \"%STAGE%\"\r\n",
            "mkdir \"%TOOLS%\"\r\n",
        ]
        for staged_file in staged_inputs:
            rel = staged_file.short_path.replace("/", "\\")
            src = staged_file.path.replace("/", "\\")
            if "\\" in rel:
                parent = rel.rsplit("\\", 1)[0]
                stage_setup.append("if not exist \"%STAGE%\\{}\" mkdir \"%STAGE%\\{}\"\r\n".format(parent, parent))
            stage_setup.append("copy /Y \"{}\" \"%STAGE%\\{}\" >nul\r\n".format(src, rel))
        for tool_file, tool_name in tool_files:
            stage_setup.append("copy /Y \"{}\" \"%TOOLS%\\{}\" >nul\r\n".format(tool_file.path.replace("/", "\\"), tool_name))
        if ctx.attr.respect_config_output and ctx.file.config:
            helper_path = helper_rel.replace("/", "\\")
            stage_setup.append("for /f \"usebackq delims=\" %%i in (`py -3 \"%EXECROOT%\\{}\" \"%STAGE%\\{}\" 2^>nul`) do set \"CONFIG_OUTPUT=%%i\"\r\n".format(helper_path, ctx.file.config.short_path.replace("/", "\\")))
            stage_setup.append("if \"%CONFIG_OUTPUT%\"==\"{}\" for /f \"usebackq delims=\" %%i in (`python \"%EXECROOT%\\{}\" \"%STAGE%\\{}\" 2^>nul`) do set \"CONFIG_OUTPUT=%%i\"\r\n".format(ctx.attr.out.replace("/", "\\"), helper_path, ctx.file.config.short_path.replace("/", "\\")))
        stage_setup.append("cd /d \"%STAGE%\"\r\n")
        wrapper_content = "".join(stage_setup) + "set \"PATH=%TOOLS%;%PATH%\"\r\n" + " ".join(quoted) + "\r\n"
        if ctx.attr.respect_config_output and ctx.file.config:
            declared_out = output_dir.path.replace("/", "\\")
            wrapper_content += (
                "if errorlevel 1 exit /b %errorlevel%\r\n" +
                "if not exist \"%STAGE%\\%CONFIG_OUTPUT%\" (echo Buffalo output directory not found: %CONFIG_OUTPUT% & exit /b 1)\r\n" +
                "if exist \"%EXECROOT%\\{}\" rmdir /s /q \"%EXECROOT%\\{}\"\r\n".format(declared_out, declared_out) +
                "mkdir \"%EXECROOT%\\{}\"\r\n".format(declared_out) +
                "xcopy /E /I /Y \"%STAGE%\\%CONFIG_OUTPUT%\\*\" \"%EXECROOT%\\{}\\\" >nul\r\n".format(declared_out) +
                "if errorlevel 2 exit /b %errorlevel%\r\n"
            )
    else:
        quoted = ["'{}'".format(part.replace("'", "'\\''")) for part in command_parts]
        for index, part in enumerate(command_parts):
            if part == "-o" and index + 1 < len(command_parts):
                out_part = command_parts[index + 1]
                if not out_part.startswith("/"):
                    quoted[index + 1] = '"$EXECROOT/{}"'.format(out_part.replace('"', '\\"'))
        stage_setup = [
            "#!/usr/bin/env sh\n",
            "set -eu\n",
            "EXECROOT=$PWD\n",
            "CONFIG_OUTPUT='{}'\n".format(ctx.attr.out.replace("'", "'\\''")),
            "STAGE=$(mktemp -d 2>/dev/null || mktemp -d -t buffalo)\n",
            "TOOLS=$STAGE/_buffalo_tools\n",
            "trap 'rm -rf \"$STAGE\"' EXIT\n",
            "mkdir -p \"$TOOLS\"\n",
        ]
        for staged_file in staged_inputs:
            rel = staged_file.short_path
            src = staged_file.path.replace("'", "'\\''")
            if "/" in rel:
                parent = rel.rsplit("/", 1)[0].replace("'", "'\\''")
                stage_setup.append("mkdir -p '$STAGE/{}'\n".format(parent))
            stage_setup.append("cp '{}' '$STAGE/{}'\n".format(src, rel.replace("'", "'\\''")))
        for tool_file, tool_name in tool_files:
            stage_setup.append("cp '{}' '$TOOLS/{}'\n".format(tool_file.path.replace("'", "'\\''"), tool_name))
            stage_setup.append("chmod +x '$TOOLS/{}'\n".format(tool_name))
        if ctx.attr.respect_config_output and ctx.file.config:
            helper_path = helper_rel.replace("'", "'\\''")
            config_rel = ctx.file.config.short_path.replace("'", "'\\''")
            stage_setup.append("if command -v python3 >/dev/null 2>&1; then CONFIG_OUTPUT=$(python3 '$EXECROOT/{}' '$STAGE/{}' 2>/dev/null || printf '%s' \"$CONFIG_OUTPUT\"); elif command -v python >/dev/null 2>&1; then CONFIG_OUTPUT=$(python '$EXECROOT/{}' '$STAGE/{}' 2>/dev/null || printf '%s' \"$CONFIG_OUTPUT\"); fi\n".format(helper_path, config_rel, helper_path, config_rel))
        stage_setup.append("cd '$STAGE'\n")
        wrapper_content = "".join(stage_setup) + "export PATH=\"$TOOLS:$PATH\"\n" + " ".join(quoted) + "\n"
        if ctx.attr.respect_config_output and ctx.file.config:
            declared_out = output_dir.path.replace("'", "'\\''")
            wrapper_content += (
                "src_dir=\"$STAGE/$CONFIG_OUTPUT\"\n" +
                "[ -d \"$src_dir\" ] || { echo \"Buffalo output directory not found: $CONFIG_OUTPUT\"; exit 1; }\n" +
                "rm -rf '$EXECROOT/{}'\n".format(declared_out) +
                "mkdir -p '$EXECROOT/{}'\n".format(declared_out) +
                "cp -R \"$src_dir\"/. '$EXECROOT/{}'/\n".format(declared_out)
            )

    ctx.actions.write(
        output = wrapper,
        content = wrapper_content,
        is_executable = True,
    )

    # Collect all inputs
    inputs = list(ctx.files.srcs)
    if ctx.file.buffalo:
        inputs.append(ctx.file.buffalo)
    if ctx.file.protoc:
        inputs.append(ctx.file.protoc)
    if ctx.file.protoc_gen_go:
        inputs.append(ctx.file.protoc_gen_go)
    if ctx.file.protoc_gen_go_grpc:
        inputs.append(ctx.file.protoc_gen_go_grpc)
    if ctx.file.protoc_gen_grpc_python:
        inputs.append(ctx.file.protoc_gen_grpc_python)
    if ctx.file.protoc_gen_prost:
        inputs.append(ctx.file.protoc_gen_prost)
    if ctx.file.protoc_gen_tonic:
        inputs.append(ctx.file.protoc_gen_tonic)
    if ctx.file.config:
        inputs.append(ctx.file.config)
    if ctx.file._config_output_reader:
        inputs.append(ctx.file._config_output_reader)
    inputs.extend(ctx.files.deps)
    inputs.append(wrapper)

    ctx.actions.run(
        outputs = [output_dir],
        inputs = inputs,
        executable = wrapper,
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
        "buffalo": attr.label(
            allow_single_file = True,
            default = Label("@buffalo_toolchain//:buffalo_bin"),
            doc = "Optional explicit path to the Buffalo binary file.",
        ),
        "protoc": attr.label(
            allow_single_file = True,
            default = Label("@buffalo_toolchain//:protoc_bin"),
            doc = "Path to the protoc binary file.",
        ),
        "protoc_gen_go": attr.label(
            allow_single_file = True,
            default = Label("@buffalo_toolchain//:protoc_gen_go_bin"),
            doc = "Path to the protoc-gen-go plugin binary file.",
        ),
        "protoc_gen_go_grpc": attr.label(
            allow_single_file = True,
            default = Label("@buffalo_toolchain//:protoc_gen_go_grpc_bin"),
            doc = "Path to the protoc-gen-go-grpc plugin binary file.",
        ),
        "protoc_gen_grpc_python": attr.label(
            allow_single_file = True,
            default = Label("@buffalo_toolchain//:protoc_gen_grpc_python_bin"),
            doc = "Path to the protoc-gen-grpc_python plugin binary file (optional in non-strict mode).",
        ),
        "protoc_gen_prost": attr.label(
            allow_single_file = True,
            default = Label("@buffalo_toolchain//:protoc_gen_prost_bin"),
            doc = "Path to the protoc-gen-prost plugin binary file. Functional only when buffalo.rust() is enabled.",
        ),
        "protoc_gen_tonic": attr.label(
            allow_single_file = True,
            default = Label("@buffalo_toolchain//:protoc_gen_tonic_bin"),
            doc = "Path to the protoc-gen-tonic plugin binary file. Functional only when buffalo.rust() is enabled.",
        ),
        "languages": attr.string_list(
            default = ["go", "python", "rust"],
            doc = "Target languages for code generation.",
        ),
        "proto_paths": attr.string_list(
            default = ["proto"],
            doc = "Directories to search for proto imports.",
        ),
        "import_paths": attr.string_list(
            default = [],
            doc = "Additional import-only directories passed to Buffalo via -I.",
        ),
        "deps": attr.label_list(
            allow_files = True,
            doc = "Additional proto file dependencies (e.g., third-party protos).",
        ),
        "out": attr.string(
            default = "gen",
            doc = "Output directory name (becomes a tree artifact).",
        ),
        "respect_config_output": attr.bool(
            default = True,
            doc = "When true and config is provided, Buffalo writes into output.base_dir from buffalo.yaml and the rule copies that directory into the declared Bazel output tree.",
        ),
        "verbose": attr.bool(
            default = False,
            doc = "Enable verbose Buffalo output.",
        ),
        "_config_output_reader": attr.label(
            allow_single_file = [".py"],
            default = Label("//buffalo:config_output_dir.py"),
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
    import_paths = [],
    copy_from_bazel_bin = False,
    compile_target = None,
    compile_out = "gen",
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
        import_paths: Additional import-only directories passed to Buffalo via -I.
        extra_args: Additional CLI arguments for `buffalo build`.
        visibility: Bazel visibility.
    """
    args = ["--config", config]
    if copy_from_bazel_bin:
        args.extend(["--copy-from-bazel-bin", compile_out])
    for lang in languages:
        args.extend(["-l", lang])
    for p in proto_paths:
        args.extend(["-p", p])
    for p in import_paths:
        args.extend(["-I", p])
    args.extend(extra_args)

    data = list(kwargs.pop("data", []))
    if compile_target:
        data.append(compile_target)

    py_binary(
        name = name,
        srcs = [Label("//buffalo:gen_runner.py")],
        main = Label("//buffalo:gen_runner.py"),
        args = args,
        data = data,
        visibility = visibility,
        **kwargs
    )
