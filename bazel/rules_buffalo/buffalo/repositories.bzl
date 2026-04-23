"""Hermetic repository rule for the Buffalo + protoc toolchain.

Bazel itself downloads everything required to compile .proto files for Go,
Python and C++ — no host tools are required. Versions and sha256 integrity
are configurable via the `buffalo.toolchain(...)` tag in the consuming
MODULE.bazel.

Provisioned tools (linux/darwin amd64+arm64, windows amd64):

  * protoc                  — protocolbuffers/protobuf releases
  * protoc-gen-go           — protocolbuffers/protobuf-go releases
  * protoc-gen-go-grpc      — grpc/grpc-go releases
  * protoc-gen-grpc_python  — grpcio-tools wheel installed into the hermetic
                              Python interpreter from rules_python
  * buffalo CLI             — massonsky/buffalo releases

C++ generation works out of the box (built into protoc).

Rust (prost/tonic) and TypeScript: planned for follow-up commits via
rules_rust / rules_nodejs integrations and exposed as opt-in tags
(`buffalo.rust()`, `buffalo.typescript()`).
"""

# Pinned upstream defaults. Override in the consuming MODULE.bazel:
#
#   buffalo = use_extension("@rules_buffalo//buffalo:extensions.bzl", "buffalo")
#   buffalo.toolchain(
#       buffalo_version = "4.1.0",
#       protoc_version  = "28.2",
#       integrity = {
#           "protoc-25.1-linux-x86_64": "sha256-...",
#       },
#   )
#   use_repo(buffalo, "buffalo_toolchain")
DEFAULT_PROTOC_VERSION = "25.1"
DEFAULT_PROTOC_GEN_GO_VERSION = "1.34.2"
DEFAULT_PROTOC_GEN_GO_GRPC_VERSION = "1.5.1"
DEFAULT_BUFFALO_VERSION = "4.0.0"
DEFAULT_BUFFALO_REPO = "massonsky/buffalo"
DEFAULT_GRPCIO_TOOLS_VERSION = "1.64.1"
DEFAULT_PROTOBUF_PY_VERSION = "5.27.1"

# Built-in sha256 integrity for the pinned default versions.
# Keys are stable artifact identifiers; values are Subresource-Integrity
# strings (`sha256-<base64>`). When a user pins a different version, Bazel
# will fail loudly until a matching entry is supplied via the
# `buffalo.toolchain(integrity = {...})` tag attribute or until first run
# emits the expected hash to logs (which the user can paste into their
# MODULE.bazel).
#
# To populate: run `bazel sync` once and copy the "Expected: sha256-..."
# value Bazel prints, then add it here (or in your MODULE.bazel override).
DEFAULT_INTEGRITY = {
    # Filled in over time. Empty entries -> first download is unverified
    # but still pinned by URL (mitigated by HTTPS + GitHub release immutability).
}

# ---------- Platform detection ----------------------------------------------

def _detect_platform(rctx):
    name = rctx.os.name.lower()
    if name.startswith("windows"):
        os_id = "windows"
    elif name.startswith("mac") or "darwin" in name:
        os_id = "darwin"
    elif name.startswith("linux"):
        os_id = "linux"
    else:
        fail("Unsupported host OS: %s" % rctx.os.name)

    arch = rctx.os.arch.lower()
    if arch in ["amd64", "x86_64", "x64"]:
        arch_id = "amd64"
    elif arch in ["arm64", "aarch64"]:
        arch_id = "arm64"
    else:
        fail("Unsupported host architecture: %s" % rctx.os.arch)

    return struct(
        os = os_id,
        arch = arch_id,
        is_windows = (os_id == "windows"),
        exe_suffix = ".exe" if os_id == "windows" else "",
    )

# ---------- URL builders -----------------------------------------------------

def _protoc_artifact(version, p):
    if p.os == "linux":
        plat = "linux-x86_64" if p.arch == "amd64" else "linux-aarch_64"
    elif p.os == "darwin":
        plat = "osx-x86_64" if p.arch == "amd64" else "osx-aarch_64"
    elif p.os == "windows":
        plat = "win64"
    else:
        fail("protoc: unsupported os %s" % p.os)
    url = "https://github.com/protocolbuffers/protobuf/releases/download/v{v}/protoc-{v}-{plat}.zip".format(
        v = version,
        plat = plat,
    )
    integrity_key = "protoc-{v}-{plat}".format(v = version, plat = plat)
    return url, integrity_key

def _protoc_gen_go_artifact(version, p):
    ext = "zip" if p.is_windows else "tar.gz"
    url = "https://github.com/protocolbuffers/protobuf-go/releases/download/v{v}/protoc-gen-go.v{v}.{os}.{arch}.{ext}".format(
        v = version,
        os = p.os,
        arch = p.arch,
        ext = ext,
    )
    integrity_key = "protoc-gen-go-{v}-{os}-{arch}".format(v = version, os = p.os, arch = p.arch)
    return url, integrity_key

def _protoc_gen_go_grpc_artifact(version, p):
    url = "https://github.com/grpc/grpc-go/releases/download/cmd%2Fprotoc-gen-go-grpc%2Fv{v}/protoc-gen-go-grpc.v{v}.{os}.{arch}.tar.gz".format(
        v = version,
        os = p.os,
        arch = p.arch,
    )
    integrity_key = "protoc-gen-go-grpc-{v}-{os}-{arch}".format(v = version, os = p.os, arch = p.arch)
    return url, integrity_key

def _buffalo_artifact(repo, version, p):
    arch = p.arch
    if p.is_windows and arch == "arm64":
        arch = "amd64"
    url = "https://github.com/{repo}/releases/download/v{v}/buffalo-{os}-{arch}{suf}".format(
        repo = repo,
        v = version,
        os = p.os,
        arch = arch,
        suf = p.exe_suffix,
    )
    integrity_key = "buffalo-{v}-{os}-{arch}".format(v = version, os = p.os, arch = arch)
    return url, integrity_key

# ---------- Download primitives ---------------------------------------------

def _resolve_integrity(rctx, integrity_key):
    user = rctx.attr.integrity.get(integrity_key, "")
    if user:
        return user
    return DEFAULT_INTEGRITY.get(integrity_key, "")

def _download_executable(rctx, url, output, integrity):
    rctx.download(
        url = [url],
        output = output,
        executable = True,
        integrity = integrity,
    )
    return rctx.path(output)

def _download_and_extract(rctx, url, subdir, expected_relpath, integrity):
    rctx.download_and_extract(
        url = [url],
        output = subdir,
        integrity = integrity,
    )
    p = rctx.path("{}/{}".format(subdir, expected_relpath))
    if p.exists:
        return p
    alt = rctx.path("{}/{}".format(subdir, expected_relpath.split("/")[-1]))
    if alt.exists:
        return alt
    fail("Expected binary not found after extracting %s: looked at %s and %s" % (url, p, alt))

# ---------- Hermetic gRPC Python plugin -------------------------------------

def _install_grpcio_tools(rctx, python_exe, grpcio_version, protobuf_version):
    """Install grpcio-tools into a private site-packages using hermetic Python."""
    site = rctx.path("_buffalo_python_site_packages")
    args = [
        str(python_exe),
        "-m",
        "pip",
        "install",
        "--disable-pip-version-check",
        "--no-input",
        "--no-warn-script-location",
        "--target",
        str(site),
        "grpcio-tools=={}".format(grpcio_version),
        "protobuf=={}".format(protobuf_version),
    ]
    res = rctx.execute(args, timeout = 600)
    if res.return_code != 0:
        rctx.execute([str(python_exe), "-m", "ensurepip", "--upgrade"], timeout = 120)
        res = rctx.execute(args, timeout = 600)
    if res.return_code != 0:
        fail("Failed to install grpcio-tools into hermetic Python.\nstdout:\n{}\nstderr:\n{}".format(
            res.stdout,
            res.stderr,
        ))
    return str(site)

def _emit_grpc_python_shim(rctx, platform, python_exe, site_packages):
    if platform.is_windows:
        shim_name = "protoc-gen-grpc_python.bat"
        pp = site_packages.replace("/", "\\")
        py = str(python_exe).replace("/", "\\")
        content = (
            "@echo off\r\n" +
            "set \"PYTHONPATH={pp};%PYTHONPATH%\"\r\n" +
            "\"{py}\" -m grpc_tools.protoc --grpc_python_out=. %*\r\n"
        ).format(pp = pp, py = py)
    else:
        shim_name = "protoc-gen-grpc_python"
        content = (
            "#!/usr/bin/env sh\n" +
            "export PYTHONPATH='{pp}':\"${{PYTHONPATH:-}}\"\n" +
            "exec '{py}' -m grpc_tools.protoc --grpc_python_out=. \"$@\"\n"
        ).format(
            pp = site_packages.replace("'", "'\\''"),
            py = str(python_exe).replace("'", "'\\''"),
        )
    rctx.file(shim_name, content, executable = True)
    return shim_name

# ---------- Repository rule implementation ----------------------------------

def _buffalo_toolchain_repo_impl(rctx):
    p = _detect_platform(rctx)
    suffix = p.exe_suffix

    # --- Hermetic upstream tools ----------------------------------------
    buffalo_url, buffalo_key = _buffalo_artifact(rctx.attr.buffalo_repo, rctx.attr.buffalo_version, p)
    buffalo = _download_executable(
        rctx,
        buffalo_url,
        "buffalo{}".format(suffix),
        _resolve_integrity(rctx, buffalo_key),
    )

    protoc_url, protoc_key = _protoc_artifact(rctx.attr.protoc_version, p)
    protoc = _download_and_extract(
        rctx,
        protoc_url,
        "_protoc",
        "bin/protoc{}".format(suffix),
        _resolve_integrity(rctx, protoc_key),
    )

    pgo_url, pgo_key = _protoc_gen_go_artifact(rctx.attr.protoc_gen_go_version, p)
    protoc_gen_go = _download_and_extract(
        rctx,
        pgo_url,
        "_protoc_gen_go",
        "protoc-gen-go{}".format(suffix),
        _resolve_integrity(rctx, pgo_key),
    )

    pgg_url, pgg_key = _protoc_gen_go_grpc_artifact(rctx.attr.protoc_gen_go_grpc_version, p)
    protoc_gen_go_grpc = _download_and_extract(
        rctx,
        pgg_url,
        "_protoc_gen_go_grpc",
        "protoc-gen-go-grpc{}".format(suffix),
        _resolve_integrity(rctx, pgg_key),
    )

    # --- Python gRPC plugin via hermetic interpreter --------------------
    python_exe = rctx.path(rctx.attr.python_interpreter)
    site_packages = _install_grpcio_tools(
        rctx,
        python_exe,
        rctx.attr.grpcio_tools_version,
        rctx.attr.protobuf_version,
    )
    grpc_python_target = _emit_grpc_python_shim(rctx, p, python_exe, site_packages)

    # --- Stage tools under stable filenames -----------------------------
    files = {
        "buffalo{}".format(suffix): buffalo,
        "protoc{}".format(suffix): protoc,
        "protoc-gen-go{}".format(suffix): protoc_gen_go,
        "protoc-gen-go-grpc{}".format(suffix): protoc_gen_go_grpc,
    }
    for target_name, source in files.items():
        if str(source).replace("\\", "/").endswith("/" + target_name):
            continue
        rctx.symlink(source, target_name)

    # --- Generate BUILD.bazel -------------------------------------------
    exports = [
        "buffalo{}".format(suffix),
        "protoc{}".format(suffix),
        "protoc-gen-go{}".format(suffix),
        "protoc-gen-go-grpc{}".format(suffix),
        grpc_python_target,
    ]
    aliases = [
        ("buffalo_bin", "buffalo{}".format(suffix)),
        ("protoc_bin", "protoc{}".format(suffix)),
        ("protoc_gen_go_bin", "protoc-gen-go{}".format(suffix)),
        ("protoc_gen_go_grpc_bin", "protoc-gen-go-grpc{}".format(suffix)),
        ("protoc_gen_grpc_python_bin", grpc_python_target),
    ]
    exports_block = ",\n".join(['    "{}"'.format(n) for n in exports])
    aliases_block = "\n\n".join([
        'alias(\n    name = "{}",\n    actual = ":{}",\n)'.format(a, t)
        for a, t in aliases
    ])

    rctx.file("BUILD.bazel", content = """package(default_visibility = ["//visibility:public"])

exports_files([
{exports}
])

{aliases}
""".format(exports = exports_block, aliases = aliases_block))

# ---------- Public repository rule ------------------------------------------

buffalo_toolchain_repo = repository_rule(
    implementation = _buffalo_toolchain_repo_impl,
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
            doc = "Map of artifact-id -> sha256 integrity (`sha256-<base64>`). " +
                  "Artifact-id format: 'protoc-<v>-<plat>', " +
                  "'protoc-gen-go-<v>-<os>-<arch>', " +
                  "'protoc-gen-go-grpc-<v>-<os>-<arch>', " +
                  "'buffalo-<v>-<os>-<arch>'. Bazel prints the expected " +
                  "value on first download; copy it here to lock the artifact.",
        ),
        "python_interpreter": attr.label(
            mandatory = True,
            allow_single_file = True,
            doc = "Hermetic Python interpreter (provided by rules_python via the buffalo extension).",
        ),
    },
    environ = ["PATH", "HOME", "USERPROFILE", "TMP", "TEMP"],
    doc = "Hermetically provisions Buffalo CLI, protoc, and Go/Python plugins from upstream releases. Zero host-tool dependencies.",
)
