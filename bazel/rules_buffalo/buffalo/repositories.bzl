"""Repository rules for locating the Buffalo binary.

Discovery order:
    1. Environment variable `BUFFALO_BIN` (absolute path to the binary).
    2. `buffalo` (or `buffalo.exe` on Windows) on the host PATH.

The repository exposes `@buffalo_toolchain//:buffalo_bin`, a `filegroup`
wrapping a small launcher script that prepends the directory containing
the real `buffalo` executable to `PATH`. This guarantees that protoc and
its plugins (`protoc-gen-go`, `protoc-gen-go-grpc`, ...) installed next
to buffalo (typically in `$GOPATH/bin`) remain discoverable inside the
Bazel sandbox.
"""

_LINUX_WRAPPER = '''#!/usr/bin/env bash
set -euo pipefail
export PATH="{buffalo_dir}:${{PATH:-}}"
exec "{buffalo}" "$@"
'''

_WINDOWS_WRAPPER = '''@echo off
set "PATH={buffalo_dir};%PATH%"
"{buffalo}" %*
'''

_BUILD_FILE = '''package(default_visibility = ["//visibility:public"])

filegroup(
    name = "buffalo_bin",
    srcs = ["{bin_name}"],
)

exports_files(["{bin_name}"])
'''

def _buffalo_toolchain_repo_impl(rctx):
    override = rctx.os.environ.get("BUFFALO_BIN", "")
    if override:
        buffalo = rctx.path(override)
        if not buffalo.exists:
            fail("BUFFALO_BIN={} does not exist".format(override))
    else:
        buffalo = rctx.which("buffalo")
        if not buffalo:
            fail(
                "Buffalo binary not found.\n" +
                "Either set BUFFALO_BIN to an absolute path, or install buffalo:\n" +
                "    go install github.com/massonsky/buffalo/cmd/buffalo@latest",
            )

    is_windows = rctx.os.name.lower().startswith("windows")
    bin_name = "buffalo.bat" if is_windows else "buffalo"
    buffalo_dir = str(buffalo.dirname)
    buffalo_path = str(buffalo)

    rctx.file("BUILD.bazel", content = _BUILD_FILE.format(bin_name = bin_name))

    if is_windows:
        wrapper = _WINDOWS_WRAPPER.format(
            buffalo_dir = buffalo_dir.replace("/", "\\"),
            buffalo = buffalo_path.replace("/", "\\"),
        )
    else:
        wrapper = _LINUX_WRAPPER.format(
            buffalo_dir = buffalo_dir,
            buffalo = buffalo_path,
        )

    rctx.file(bin_name, content = wrapper, executable = True)

buffalo_toolchain_repo = repository_rule(
    implementation = _buffalo_toolchain_repo_impl,
    environ = ["PATH", "BUFFALO_BIN"],
    local = True,
    doc = "Locates the buffalo binary on the host system and exposes it as @buffalo_toolchain//:buffalo_bin.",
)
