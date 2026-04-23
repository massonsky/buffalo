"""Hermetic repository rule for the Buffalo + protoc toolchain.

Bazel itself downloads everything required to compile .proto files for Go,
Python, C++ and (opt-in) Rust — no host tools are required. Versions and
sha256 integrity are configurable via the `buffalo.toolchain(...)` and
`buffalo.rust(...)` tags in the consuming MODULE.bazel.

Provisioned tools (linux/darwin amd64+arm64, windows amd64):

  * protoc                  — protocolbuffers/protobuf releases
  * protoc-gen-go           — protocolbuffers/protobuf-go releases
  * protoc-gen-go-grpc      — grpc/grpc-go releases
  * protoc-gen-grpc_python  — grpcio-tools wheel installed into the hermetic
                              Python interpreter from rules_python
  * buffalo CLI             — massonsky/buffalo releases

Optional (enabled by `buffalo.rust()` tag):

  * protoc-gen-prost        — neoeinstein/protoc-gen-prost releases
  * protoc-gen-tonic        — neoeinstein/protoc-gen-prost releases

C++ generation works out of the box (built into protoc).
TypeScript: planned for a follow-up commit via `aspect_rules_js`.
"""

# Pinned upstream defaults.
DEFAULT_PROTOC_VERSION = "25.1"
DEFAULT_PROTOC_GEN_GO_VERSION = "1.34.2"
DEFAULT_PROTOC_GEN_GO_GRPC_VERSION = "1.5.1"
DEFAULT_BUFFALO_VERSION = "4.0.0"
DEFAULT_BUFFALO_REPO = "massonsky/buffalo"
DEFAULT_GRPCIO_TOOLS_VERSION = "1.64.1"
DEFAULT_PROTOBUF_PY_VERSION = "5.27.1"

DEFAULT_PROTOC_GEN_PROST_VERSION = "0.4.0"
DEFAULT_PROTOC_GEN_TONIC_VERSION = "0.4.1"
DEFAULT_PROTOC_GEN_PROST_REPO = "neoeinstein/protoc-gen-prost"

DEFAULT_NODE_VERSION = "20.18.0"
DEFAULT_TS_PROTO_VERSION = "1.181.2"

DEFAULT_INTEGRITY = {}

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

    # Rust target triple for prebuilt prost/tonic plugins.
    if os_id == "linux":
        triple = "x86_64-unknown-linux-gnu" if arch_id == "amd64" else "aarch64-unknown-linux-gnu"
    elif os_id == "darwin":
        triple = "x86_64-apple-darwin" if arch_id == "amd64" else "aarch64-apple-darwin"
    elif os_id == "windows":
        triple = "x86_64-pc-windows-msvc"
    else:
        triple = ""

    return struct(
        os = os_id,
        arch = arch_id,
        is_windows = (os_id == "windows"),
        exe_suffix = ".exe" if os_id == "windows" else "",
        rust_triple = triple,
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
    return url, "protoc-{v}-{plat}".format(v = version, plat = plat)

def _protoc_gen_go_artifact(version, p):
    ext = "zip" if p.is_windows else "tar.gz"
    url = "https://github.com/protocolbuffers/protobuf-go/releases/download/v{v}/protoc-gen-go.v{v}.{os}.{arch}.{ext}".format(
        v = version,
        os = p.os,
        arch = p.arch,
        ext = ext,
    )
    return url, "protoc-gen-go-{v}-{os}-{arch}".format(v = version, os = p.os, arch = p.arch)

def _protoc_gen_go_grpc_artifact(version, p):
    url = "https://github.com/grpc/grpc-go/releases/download/cmd%2Fprotoc-gen-go-grpc%2Fv{v}/protoc-gen-go-grpc.v{v}.{os}.{arch}.tar.gz".format(
        v = version,
        os = p.os,
        arch = p.arch,
    )
    return url, "protoc-gen-go-grpc-{v}-{os}-{arch}".format(v = version, os = p.os, arch = p.arch)

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
    return url, "buffalo-{v}-{os}-{arch}".format(v = version, os = p.os, arch = arch)

def _rust_plugin_artifact(repo, plugin_name, version, p):
    if not p.rust_triple:
        fail("Unsupported platform for Rust plugin %s: %s/%s" % (plugin_name, p.os, p.arch))
    ext = "zip" if p.is_windows else "tar.xz"
    url = "https://github.com/{repo}/releases/download/{name}-v{v}/{name}-v{v}-{triple}.{ext}".format(
        repo = repo,
        name = plugin_name,
        v = version,
        triple = p.rust_triple,
        ext = ext,
    )
    return url, "{name}-{v}-{triple}".format(name = plugin_name, v = version, triple = p.rust_triple)

def _node_artifact(version, p):
    """Returns (url, integrity_key, archive_dir, node_relpath, npm_relpath)."""
    if p.os == "linux":
        plat = "linux-x64" if p.arch == "amd64" else "linux-arm64"
        ext = "tar.xz"
    elif p.os == "darwin":
        plat = "darwin-x64" if p.arch == "amd64" else "darwin-arm64"
        ext = "tar.xz"
    elif p.os == "windows":
        plat = "win-x64"
        ext = "zip"
    else:
        fail("node: unsupported os %s" % p.os)

    archive_basename = "node-v{v}-{plat}".format(v = version, plat = plat)
    url = "https://nodejs.org/dist/v{v}/{base}.{ext}".format(v = version, base = archive_basename, ext = ext)
    if p.is_windows:
        node_rel = "{}/node.exe".format(archive_basename)
        npm_rel = "{}/npm.cmd".format(archive_basename)
    else:
        node_rel = "{}/bin/node".format(archive_basename)
        npm_rel = "{}/bin/npm".format(archive_basename)
    return struct(
        url = url,
        integrity_key = "node-{v}-{plat}".format(v = version, plat = plat),
        archive_dir = archive_basename,
        node_relpath = node_rel,
        npm_relpath = npm_rel,
    )

# ---------- Hermetic TypeScript plugin --------------------------------------

def _provision_node(rctx, p, version):
    spec = _node_artifact(version, p)
    rctx.download_and_extract(
        url = [spec.url],
        output = "_node",
        integrity = _resolve_integrity(rctx, spec.integrity_key),
    )
    node_path = rctx.path("_node/{}".format(spec.node_relpath))
    npm_path = rctx.path("_node/{}".format(spec.npm_relpath))
    if not node_path.exists:
        fail("Node binary not found after extracting %s: %s" % (spec.url, node_path))
    if not npm_path.exists:
        fail("npm binary not found after extracting %s: %s" % (spec.url, npm_path))
    return struct(node = node_path, npm = npm_path, archive_dir = spec.archive_dir)

def _install_ts_proto(rctx, p, node_info, ts_proto_version):
    """Install ts-proto into a private node_modules using hermetic Node."""
    project_dir = rctx.path("_buffalo_ts_proto")
    rctx.file(
        "_buffalo_ts_proto/package.json",
        "{{\n  \"name\": \"buffalo-ts-proto-host\",\n  \"version\": \"0.0.0\",\n  \"private\": true,\n  \"dependencies\": {{\n    \"ts-proto\": \"{}\"\n  }}\n}}\n".format(ts_proto_version),
        executable = False,
    )

    npm_cmd = str(node_info.npm)
    if p.is_windows:
        # On Windows npm is a .cmd; rctx.execute can run it directly.
        args = [npm_cmd, "install", "--no-audit", "--no-fund", "--no-progress", "--loglevel=error"]
    else:
        # Use node to run npm-cli.js explicitly to avoid PATH/env issues.
        npm_cli = rctx.path("_node/{}/lib/node_modules/npm/bin/npm-cli.js".format(node_info.archive_dir))
        args = [str(node_info.node), str(npm_cli), "install", "--no-audit", "--no-fund", "--no-progress", "--loglevel=error"]

    res = rctx.execute(args, working_directory = str(project_dir), timeout = 600)
    if res.return_code != 0:
        fail("Failed to install ts-proto into hermetic Node.\nstdout:\n{}\nstderr:\n{}".format(
            res.stdout,
            res.stderr,
        ))

    plugin_js = rctx.path("_buffalo_ts_proto/node_modules/ts-proto/protoc-gen-ts_proto")
    if not plugin_js.exists:
        # ts-proto >=1.x ships the binary as protoc-gen-ts_proto.js in build/
        plugin_js = rctx.path("_buffalo_ts_proto/node_modules/ts-proto/build/plugin.js")
    if not plugin_js.exists:
        fail("ts-proto plugin entrypoint not found in installed package.")
    return plugin_js

def _emit_ts_proto_shim(rctx, p, node_path, plugin_js):
    if p.is_windows:
        shim_name = "protoc-gen-ts_proto.bat"
        node = str(node_path).replace("/", "\\")
        plugin = str(plugin_js).replace("/", "\\")
        content = (
            "@echo off\r\n" +
            "\"{node}\" \"{plugin}\" %*\r\n"
        ).format(node = node, plugin = plugin)
    else:
        shim_name = "protoc-gen-ts_proto"
        content = (
            "#!/usr/bin/env sh\n" +
            "exec '{node}' '{plugin}' \"$@\"\n"
        ).format(
            node = str(node_path).replace("'", "'\\''"),
            plugin = str(plugin_js).replace("'", "'\\''"),
        )
    rctx.file(shim_name, content, executable = True)
    return shim_name

# ---------- Download primitives ---------------------------------------------

def _resolve_integrity(rctx, integrity_key):
    user = rctx.attr.integrity.get(integrity_key, "")
    if user:
        return user
    return DEFAULT_INTEGRITY.get(integrity_key, "")

def _download_executable(rctx, url, output, integrity):
    rctx.download(url = [url], output = output, executable = True, integrity = integrity)
    return rctx.path(output)

def _download_and_extract(rctx, url, subdir, expected_relpath, integrity):
    rctx.download_and_extract(url = [url], output = subdir, integrity = integrity)
    p = rctx.path("{}/{}".format(subdir, expected_relpath))
    if p.exists:
        return p
    alt = rctx.path("{}/{}".format(subdir, expected_relpath.split("/")[-1]))
    if alt.exists:
        return alt
    fail("Expected binary not found after extracting %s: looked at %s and %s" % (url, p, alt))

# ---------- Hermetic gRPC Python plugin -------------------------------------

def _install_grpcio_tools(rctx, python_exe, grpcio_version, protobuf_version):
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

# ---------- Stub generation -------------------------------------------------

def _emit_disabled_stub(rctx, platform, name, reason):
    """Emit a non-functional placeholder for plugins that weren't enabled."""
    suffix = ".bat" if platform.is_windows else ""
    target = "{}{}".format(name, suffix)
    if platform.is_windows:
        content = "@echo off\r\necho {}\r\nexit /b 1\r\n".format(reason)
    else:
        content = "#!/usr/bin/env sh\necho '{}' >&2\nexit 1\n".format(reason.replace("'", "'\\''"))
    rctx.file(target, content, executable = True)
    return target

# ---------- Repository rule implementation ----------------------------------

def _buffalo_toolchain_repo_impl(rctx):
    p = _detect_platform(rctx)
    suffix = p.exe_suffix

    # --- Hermetic upstream tools (always provisioned) ------------------
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

    python_exe = rctx.path(rctx.attr.python_interpreter)
    site_packages = _install_grpcio_tools(
        rctx,
        python_exe,
        rctx.attr.grpcio_tools_version,
        rctx.attr.protobuf_version,
    )
    grpc_python_target = _emit_grpc_python_shim(rctx, p, python_exe, site_packages)

    # --- Optional: Rust plugins (opt-in via buffalo.rust() tag) --------
    prost_target = None
    tonic_target = None
    if rctx.attr.enable_rust:
        prost_url, prost_key = _rust_plugin_artifact(
            rctx.attr.protoc_gen_prost_repo,
            "protoc-gen-prost",
            rctx.attr.protoc_gen_prost_version,
            p,
        )
        prost = _download_and_extract(
            rctx,
            prost_url,
            "_protoc_gen_prost",
            "protoc-gen-prost{}".format(suffix),
            _resolve_integrity(rctx, prost_key),
        )
        prost_target = "protoc-gen-prost{}".format(suffix)
        rctx.symlink(prost, prost_target)

        tonic_url, tonic_key = _rust_plugin_artifact(
            rctx.attr.protoc_gen_prost_repo,
            "protoc-gen-tonic",
            rctx.attr.protoc_gen_tonic_version,
            p,
        )
        tonic = _download_and_extract(
            rctx,
            tonic_url,
            "_protoc_gen_tonic",
            "protoc-gen-tonic{}".format(suffix),
            _resolve_integrity(rctx, tonic_key),
        )
        tonic_target = "protoc-gen-tonic{}".format(suffix)
        rctx.symlink(tonic, tonic_target)
    else:
        prost_target = _emit_disabled_stub(
            rctx,
            p,
            "protoc-gen-prost",
            "protoc-gen-prost is disabled. Add buffalo.rust() to MODULE.bazel to enable.",
        )
        tonic_target = _emit_disabled_stub(
            rctx,
            p,
            "protoc-gen-tonic",
            "protoc-gen-tonic is disabled. Add buffalo.rust() to MODULE.bazel to enable.",
        )

    # --- Optional: TypeScript plugin (opt-in via buffalo.typescript() tag)
    ts_proto_target = None
    if rctx.attr.enable_typescript:
        node_info = _provision_node(rctx, p, rctx.attr.node_version)
        plugin_js = _install_ts_proto(rctx, p, node_info, rctx.attr.ts_proto_version)
        ts_proto_target = _emit_ts_proto_shim(rctx, p, node_info.node, plugin_js)
    else:
        ts_proto_target = _emit_disabled_stub(
            rctx,
            p,
            "protoc-gen-ts_proto",
            "protoc-gen-ts_proto is disabled. Add buffalo.typescript() to MODULE.bazel to enable.",
        )

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
        prost_target,
        tonic_target,
        ts_proto_target,
    ]
    aliases = [
        ("buffalo_bin", "buffalo{}".format(suffix)),
        ("protoc_bin", "protoc{}".format(suffix)),
        ("protoc_gen_go_bin", "protoc-gen-go{}".format(suffix)),
        ("protoc_gen_go_grpc_bin", "protoc-gen-go-grpc{}".format(suffix)),
        ("protoc_gen_grpc_python_bin", grpc_python_target),
        ("protoc_gen_prost_bin", prost_target),
        ("protoc_gen_tonic_bin", tonic_target),
        ("protoc_gen_ts_proto_bin", ts_proto_target),
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
        "enable_rust": attr.bool(default = False),
        "protoc_gen_prost_version": attr.string(default = DEFAULT_PROTOC_GEN_PROST_VERSION),
        "protoc_gen_tonic_version": attr.string(default = DEFAULT_PROTOC_GEN_TONIC_VERSION),
        "protoc_gen_prost_repo": attr.string(default = DEFAULT_PROTOC_GEN_PROST_REPO),
        "enable_typescript": attr.bool(default = False),
        "node_version": attr.string(default = DEFAULT_NODE_VERSION),
        "ts_proto_version": attr.string(default = DEFAULT_TS_PROTO_VERSION),
        "integrity": attr.string_dict(
            default = {},
            doc = "Map of artifact-id -> sha256 integrity (`sha256-<base64>`). " +
                  "Artifact-id format: 'protoc-<v>-<plat>', " +
                  "'protoc-gen-go-<v>-<os>-<arch>', " +
                  "'protoc-gen-go-grpc-<v>-<os>-<arch>', " +
                  "'buffalo-<v>-<os>-<arch>', " +
                  "'protoc-gen-prost-<v>-<triple>', " +
                  "'protoc-gen-tonic-<v>-<triple>', " +
                  "'node-<v>-<plat>'.",
        ),
        "python_interpreter": attr.label(
            mandatory = True,
            allow_single_file = True,
            doc = "Hermetic Python interpreter (provided by rules_python via the buffalo extension).",
        ),
    },
    environ = ["PATH", "HOME", "USERPROFILE", "TMP", "TEMP"],
    doc = "Hermetically provisions Buffalo CLI, protoc, Go/Python plugins and (opt-in) Rust plugins from upstream releases.",
)
