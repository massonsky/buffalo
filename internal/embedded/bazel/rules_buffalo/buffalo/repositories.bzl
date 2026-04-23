"""Repository rules for locating Buffalo and protoc tool binaries."""

def _require_tool(rctx, name, install_hint):
    tool = rctx.which(name)
    if not tool:
        fail("{} binary not found in PATH.\n{}".format(name, install_hint))
    return tool

def _find_grpc_tools_python(rctx, is_windows):
    python_candidates = []

    python = rctx.which("python")
    if python:
        python_candidates.append([str(python)])

    python3 = rctx.which("python3")
    if python3 and str(python3) != str(python):
        python_candidates.append([str(python3)])

    if is_windows:
        py_launcher = rctx.which("py")
        if py_launcher:
            py_launcher_path = str(py_launcher)
            if py_launcher_path.lower().endswith("\\py.exe") or py_launcher_path.lower().endswith("/py.exe"):
                python_candidates.append([py_launcher_path, "-3"])
            else:
                python_candidates.append([py_launcher_path])

    if not python_candidates:
        return None

    last_error = ""
    for base_args in python_candidates:
        args = list(base_args)
        args.extend([
            "-c",
            "import grpc_tools.protoc",
        ])

        result = rctx.execute(args)
        if result.return_code == 0:
            return base_args
        else:
            last_error = "{}\n{}".format(" ".join(base_args), result.stderr)

    return None

def _buffalo_toolchain_repo_impl(rctx):
    is_windows = rctx.os.name.lower().startswith("windows")
    suffix = ".exe" if is_windows else ""
    protoc_name = "protoc.bat" if is_windows else "protoc"

    buffalo = _require_tool(
        rctx,
        "buffalo",
        "Install: go install github.com/massonsky/buffalo/cmd/buffalo@latest",
    )
    protoc_gen_go = _require_tool(
        rctx,
        "protoc-gen-go",
        "Install: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest",
    )
    protoc_gen_go_grpc = _require_tool(
        rctx,
        "protoc-gen-go-grpc",
        "Install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
    )
    grpc_tools_python = _find_grpc_tools_python(rctx, is_windows)

    files = {
        "buffalo{}".format(suffix): buffalo,
        "protoc-gen-go{}".format(suffix): protoc_gen_go,
        "protoc-gen-go-grpc{}".format(suffix): protoc_gen_go_grpc,
    }

    for target_name, source in files.items():
        rctx.symlink(source, target_name)

    if grpc_tools_python:
        if is_windows:
            extra_args = " ".join(grpc_tools_python[1:])
            protoc_content = "@echo off\r\n\"{}\" {} -m grpc_tools.protoc %*\r\n".format(
                grpc_tools_python[0],
                extra_args,
            )
        else:
            extra_args = " ".join(grpc_tools_python[1:])
            protoc_content = "#!/usr/bin/env sh\nexec \"{}\" {} -m grpc_tools.protoc \"$@\"\n".format(
                grpc_tools_python[0],
                extra_args,
            )
        rctx.file(protoc_name, protoc_content, executable = True)
    else:
        protoc = _require_tool(
            rctx,
            "protoc",
            "Install Protocol Buffers compiler and expose `protoc` in PATH.",
        )
        grpc_python_plugin = _require_tool(
            rctx,
            "protoc-gen-grpc_python",
            "Install grpcio-tools in user/site Python or expose protoc-gen-grpc_python in PATH.",
        )
        rctx.symlink(protoc, protoc_name)
        rctx.symlink(grpc_python_plugin, "protoc-gen-grpc_python{}".format(suffix))

    rctx.file("BUILD.bazel", content = """\
package(default_visibility = ["//visibility:public"])

exports_files([
    "{buffalo}",
    "{protoc}",
    "{protoc_gen_go}",
    "{protoc_gen_go_grpc}",
])

alias(
    name = "buffalo_bin",
    actual = ":{buffalo}",
)

alias(
    name = "protoc_bin",
    actual = ":{protoc}",
)

alias(
    name = "protoc_gen_go_bin",
    actual = ":{protoc_gen_go}",
)

alias(
    name = "protoc_gen_go_grpc_bin",
    actual = ":{protoc_gen_go_grpc}",
)
""".format(
        buffalo = "buffalo{}".format(suffix),
        protoc = protoc_name,
        protoc_gen_go = "protoc-gen-go{}".format(suffix),
        protoc_gen_go_grpc = "protoc-gen-go-grpc{}".format(suffix),
    ))

buffalo_toolchain_repo = repository_rule(
    implementation = _buffalo_toolchain_repo_impl,
    environ = ["PATH"],
    doc = "Locates Buffalo and required protoc-related binaries on the system PATH and creates a repository with them.",
)
