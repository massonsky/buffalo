"""Repository rules for locating Buffalo and protoc tool binaries."""

_BUFFALO_GO_TOOLS = {
    "buffalo": "github.com/massonsky/buffalo/cmd/buffalo@latest",
    "protoc-gen-go": "google.golang.org/protobuf/cmd/protoc-gen-go@latest",
    "protoc-gen-go-grpc": "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
}

def _require_tool(rctx, name, install_hint):
    tool = rctx.which(name)
    if not tool:
        fail("{} binary not found in PATH.\n{}".format(name, install_hint))
    return tool

def _python_candidates(rctx, is_windows):
    candidates = []

    python = rctx.which("python")
    if python:
        candidates.append([str(python)])

    python3 = rctx.which("python3")
    if python3 and str(python3) != str(python):
        candidates.append([str(python3)])

    if is_windows:
        py_launcher = rctx.which("py")
        if py_launcher:
            py_launcher_path = str(py_launcher)
            if py_launcher_path.lower().endswith("\\py.exe") or py_launcher_path.lower().endswith("/py.exe"):
                candidates.append([py_launcher_path, "-3"])
            else:
                candidates.append([py_launcher_path])

    return candidates

def _run_or_fail(rctx, args, error_message, environment = None):
    result = rctx.execute(args, environment = environment)
    if result.return_code != 0:
        details = []
        if result.stdout:
            details.append("stdout:\n{}".format(result.stdout))
        if result.stderr:
            details.append("stderr:\n{}".format(result.stderr))
        fail("{}\ncommand: {}\n{}".format(error_message, " ".join(args), "\n".join(details)))
    return result

def _install_go_tool(rctx, name, module, suffix):
    go = rctx.which("go")
    if not go:
        fail("{} binary not found in PATH and Go is unavailable for auto-install.\nInstall {} or add `go` to PATH.".format(name, module))

    gobin = str(rctx.path("_buffalo_tools_bin"))
    env = {
        "GOBIN": gobin,
        "GOCACHE": str(rctx.path("_buffalo_go_cache")),
        "GOMODCACHE": str(rctx.path("_buffalo_go_modcache")),
        "CGO_ENABLED": "0",
    }

    _run_or_fail(
        rctx,
        [str(go), "install", module],
        "Failed to auto-install required tool '{}' via `go install`.".format(name),
        environment = env,
    )

    installed = rctx.path("_buffalo_tools_bin/{}{}".format(name, suffix))
    if not installed.exists:
        fail("Failed to install {} into Bazel local toolchain directory: {}".format(name, installed))
    return installed

def _install_grpc_tools_python(rctx, is_windows):
    candidates = _python_candidates(rctx, is_windows)
    if not candidates:
        fail("python/python3/py not found in PATH. Python is required to bootstrap grpcio-tools into Bazel sandbox toolchain.")

    site_packages = str(rctx.path("_buffalo_python_site_packages"))
    last_error = ""

    for base_args in candidates:
        install_args = list(base_args)
        install_args.extend([
            "-m",
            "pip",
            "install",
            "--disable-pip-version-check",
            "--no-input",
            "--target",
            site_packages,
            "grpcio-tools",
            "protobuf",
        ])

        result = rctx.execute(install_args)
        if result.return_code == 0:
            probe_args = list(base_args)
            probe_args.extend(["-c", "import grpc_tools.protoc"])
            probe = rctx.execute(probe_args, environment = {"PYTHONPATH": site_packages})
            if probe.return_code == 0:
                return struct(
                    python = base_args,
                    site_packages = site_packages,
                )

        ensurepip_args = list(base_args)
        ensurepip_args.extend(["-m", "ensurepip", "--upgrade"])
        _ = rctx.execute(ensurepip_args)

        result = rctx.execute(install_args)
        if result.return_code == 0:
            probe_args = list(base_args)
            probe_args.extend(["-c", "import grpc_tools.protoc"])
            probe = rctx.execute(probe_args, environment = {"PYTHONPATH": site_packages})
            if probe.return_code == 0:
                return struct(
                    python = base_args,
                    site_packages = site_packages,
                )

        last_error = "{}\n{}".format(" ".join(base_args), result.stderr)

    fail("Failed to bootstrap grpcio-tools/protobuf for Bazel Buffalo sandbox.\n{}".format(last_error))

def _buffalo_toolchain_repo_impl(rctx):
    is_windows = rctx.os.name.lower().startswith("windows")
    suffix = ".exe" if is_windows else ""
    protoc_name = "protoc.bat" if is_windows else "protoc"

    buffalo = _install_go_tool(rctx, "buffalo", _BUFFALO_GO_TOOLS["buffalo"], suffix)
    protoc_gen_go = _install_go_tool(rctx, "protoc-gen-go", _BUFFALO_GO_TOOLS["protoc-gen-go"], suffix)
    protoc_gen_go_grpc = _install_go_tool(rctx, "protoc-gen-go-grpc", _BUFFALO_GO_TOOLS["protoc-gen-go-grpc"], suffix)
    grpc_tools_python = _install_grpc_tools_python(rctx, is_windows)

    files = {
        "buffalo{}".format(suffix): buffalo,
        "protoc-gen-go{}".format(suffix): protoc_gen_go,
        "protoc-gen-go-grpc{}".format(suffix): protoc_gen_go_grpc,
    }

    for target_name, source in files.items():
        rctx.symlink(source, target_name)

    py_cmd = grpc_tools_python.python
    pythonpath = grpc_tools_python.site_packages
    if is_windows:
        extra_args = " ".join(py_cmd[1:])
        pythonpath_line = ""
        if pythonpath:
            pythonpath_line = "set \"PYTHONPATH={}{}%PYTHONPATH%\"\r\n".format(pythonpath.replace("/", "\\"), ";")
        protoc_content = "@echo off\r\n{}\"{}\" {} -m grpc_tools.protoc %*\r\n".format(
            pythonpath_line,
            py_cmd[0],
            extra_args,
        )
    else:
        extra_args = " ".join(py_cmd[1:])
        pythonpath_line = ""
        if pythonpath:
            escaped = pythonpath.replace("'", "'\\''")
            pythonpath_line = "export PYTHONPATH='{}':\"${{PYTHONPATH:-}}\"\n".format(escaped)
        protoc_content = "#!/usr/bin/env sh\n{}exec \"{}\" {} -m grpc_tools.protoc \"$@\"\n".format(
            pythonpath_line,
            py_cmd[0],
            extra_args,
        )
    rctx.file(protoc_name, protoc_content, executable = True)

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
    environ = ["PATH", "HOME", "USERPROFILE", "TMP", "TEMP"],
    doc = "Bootstraps Buffalo and required protoc tooling into the Bazel toolchain repository for sandbox execution.",
)
