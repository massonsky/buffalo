"""Hermetic repository rules for Buffalo + protoc toolchain.

Goal: zero host-tool dependencies for protoc and Go-side plugins. Bazel
itself downloads pinned, prebuilt binaries from upstream releases:

  * protoc                 -> protocolbuffers/protobuf releases
  * protoc-gen-go          -> protocolbuffers/protobuf-go releases
  * protoc-gen-go-grpc     -> grpc/grpc-go releases
  * buffalo CLI            -> massonsky/buffalo releases (configurable)

The Python gRPC plugin (`protoc-gen-grpc_python`) and Rust plugins
(`protoc-gen-prost`, `protoc-gen-tonic`) currently still require host
Python / cargo as a fallback. They are tracked for full hermetic
support in the next iterations (rules_python pip / rules_rust cargo).

Override knobs (env vars):

  Versions:
    BUFFALO_TOOLCHAIN_PROTOC_VERSION
    BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_VERSION
    BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_GRPC_VERSION
    BUFFALO_TOOLCHAIN_BUFFALO_VERSION
    BUFFALO_TOOLCHAIN_BUFFALO_REPO   (default: "massonsky/buffalo")

  Direct URL overrides (skip auto-pick):
    BUFFALO_TOOLCHAIN_BUFFALO_URL
    BUFFALO_TOOLCHAIN_PROTOC_URL
    BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_URL
    BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_GRPC_URL
    BUFFALO_TOOLCHAIN_PROTOC_GEN_GRPC_PYTHON_URL
    BUFFALO_TOOLCHAIN_PROTOC_GEN_PROST_URL
    BUFFALO_TOOLCHAIN_PROTOC_GEN_TONIC_URL

  Mode:
    BUFFALO_TOOLCHAIN_STRICT_SANDBOX=1  -> fail if any URL cannot be
                                           determined (no host fallback
                                           even for python/rust).
"""

# ---------------------------------------------------------------------------
# Pinned default versions. Override via env if you need a different release.
# ---------------------------------------------------------------------------

_DEFAULT_PROTOC_VERSION = "25.1"
_DEFAULT_PROTOC_GEN_GO_VERSION = "1.34.2"
_DEFAULT_PROTOC_GEN_GO_GRPC_VERSION = "1.5.1"
_DEFAULT_BUFFALO_VERSION = "4.0.0"
_DEFAULT_BUFFALO_REPO = "massonsky/buffalo"

# ---------------------------------------------------------------------------
# Env var names.
# ---------------------------------------------------------------------------

_STRICT_MODE_ENV = "BUFFALO_TOOLCHAIN_STRICT_SANDBOX"

_VERSION_ENVS = {
    "protoc": "BUFFALO_TOOLCHAIN_PROTOC_VERSION",
    "protoc-gen-go": "BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_VERSION",
    "protoc-gen-go-grpc": "BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_GRPC_VERSION",
    "buffalo": "BUFFALO_TOOLCHAIN_BUFFALO_VERSION",
}

_BUFFALO_REPO_ENV = "BUFFALO_TOOLCHAIN_BUFFALO_REPO"

_TOOL_URL_ENVS = {
    "buffalo": "BUFFALO_TOOLCHAIN_BUFFALO_URL",
    "protoc": "BUFFALO_TOOLCHAIN_PROTOC_URL",
    "protoc-gen-go": "BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_URL",
    "protoc-gen-go-grpc": "BUFFALO_TOOLCHAIN_PROTOC_GEN_GO_GRPC_URL",
    "protoc-gen-grpc_python": "BUFFALO_TOOLCHAIN_PROTOC_GEN_GRPC_PYTHON_URL",
    "protoc-gen-prost": "BUFFALO_TOOLCHAIN_PROTOC_GEN_PROST_URL",
    "protoc-gen-tonic": "BUFFALO_TOOLCHAIN_PROTOC_GEN_TONIC_URL",
}

# ---------------------------------------------------------------------------
# Small helpers.
# ---------------------------------------------------------------------------

def _env(rctx, key, default = ""):
    return rctx.os.environ.get(key, default)

def _strict_mode(rctx):
    return _env(rctx, _STRICT_MODE_ENV, "0").lower() in ["1", "true", "yes", "on"]

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

# ---------------------------------------------------------------------------
# Platform detection.
# ---------------------------------------------------------------------------

def _detect_platform(rctx):
    """Return a struct describing the host platform.

    Fields:
      os         -> "linux" | "darwin" | "windows"
      arch       -> "amd64" | "arm64"
      is_windows -> bool
      exe_suffix -> "" | ".exe"
    """
    name = rctx.os.name.lower()
    if name.startswith("windows"):
        os_id = "windows"
    elif name.startswith("mac") or "darwin" in name:
        os_id = "darwin"
    elif name.startswith("linux"):
        os_id = "linux"
    else:
        fail("Unsupported host OS for Buffalo toolchain: {}".format(rctx.os.name))

    arch_raw = rctx.os.arch.lower()
    if arch_raw in ["amd64", "x86_64", "x64"]:
        arch_id = "amd64"
    elif arch_raw in ["arm64", "aarch64"]:
        arch_id = "arm64"
    else:
        fail("Unsupported host architecture for Buffalo toolchain: {}".format(rctx.os.arch))

    return struct(
        os = os_id,
        arch = arch_id,
        is_windows = (os_id == "windows"),
        exe_suffix = ".exe" if os_id == "windows" else "",
    )

# ---------------------------------------------------------------------------
# URL builders for upstream prebuilt releases.
# ---------------------------------------------------------------------------

def _protoc_url(version, platform):
    """https://github.com/protocolbuffers/protobuf/releases"""
    if platform.os == "linux":
        plat = "linux-x86_64" if platform.arch == "amd64" else "linux-aarch_64"
    elif platform.os == "darwin":
        plat = "osx-x86_64" if platform.arch == "amd64" else "osx-aarch_64"
    elif platform.os == "windows":
        # Official protoc has no win-arm64; win64 (amd64) is what Microsoft
        # emulates on ARM64 Windows.
        plat = "win64"
    else:
        fail("protoc: unsupported os {}".format(platform.os))
    return "https://github.com/protocolbuffers/protobuf/releases/download/v{v}/protoc-{v}-{p}.zip".format(
        v = version,
        p = plat,
    )

def _protoc_gen_go_url(version, platform):
    """https://github.com/protocolbuffers/protobuf-go/releases"""
    ext = "zip" if platform.is_windows else "tar.gz"
    return "https://github.com/protocolbuffers/protobuf-go/releases/download/v{v}/protoc-gen-go.v{v}.{os}.{arch}.{ext}".format(
        v = version,
        os = platform.os,
        arch = platform.arch,
        ext = ext,
    )

def _protoc_gen_go_grpc_url(version, platform):
    """https://github.com/grpc/grpc-go/releases (always tar.gz, all platforms)."""
    return "https://github.com/grpc/grpc-go/releases/download/cmd%2Fprotoc-gen-go-grpc%2Fv{v}/protoc-gen-go-grpc.v{v}.{os}.{arch}.tar.gz".format(
        v = version,
        os = platform.os,
        arch = platform.arch,
    )

def _buffalo_url(repo, version, platform):
    # Releases publish raw binaries (no archive). Windows arm64 not built;
    # fall back to amd64 (works under Microsoft x64 emulation).
    arch = platform.arch
    if platform.is_windows and arch == "arm64":
        arch = "amd64"
    name = "buffalo-{os}-{arch}{suf}".format(
        os = platform.os,
        arch = arch,
        suf = platform.exe_suffix,
    )
    return "https://github.com/{repo}/releases/download/v{v}/{name}".format(
        repo = repo,
        v = version,
        name = name,
    )

# ---------------------------------------------------------------------------
# Download primitives.
# ---------------------------------------------------------------------------

def _download_executable(rctx, url, output_name):
    """Download a single executable file (no archive) to the repo root."""
    rctx.download(
        url = [url],
        output = output_name,
        executable = True,
    )
    return rctx.path(output_name)

def _download_and_extract_tool(rctx, url, subdir, expected_relpath):
    """Download an archive and return rctx.path of `subdir/expected_relpath`."""
    rctx.download_and_extract(
        url = [url],
        output = subdir,
    )
    p = rctx.path("{}/{}".format(subdir, expected_relpath))
    if not p.exists:
        # Some archives place the binary directly at the root, not in bin/.
        alt = rctx.path("{}/{}".format(subdir, expected_relpath.split("/")[-1]))
        if alt.exists:
            return alt
        fail("Expected binary not found after extracting {}: looked at {} and {}".format(url, p, alt))
    return p

# ---------------------------------------------------------------------------
# Hermetic provisioning of upstream tools.
# ---------------------------------------------------------------------------

def _provision_buffalo(rctx, platform):
    url = _env(rctx, _TOOL_URL_ENVS["buffalo"])
    if not url:
        version = _env(rctx, _VERSION_ENVS["buffalo"], _DEFAULT_BUFFALO_VERSION)
        repo = _env(rctx, _BUFFALO_REPO_ENV, _DEFAULT_BUFFALO_REPO)
        url = _buffalo_url(repo, version, platform)
    output = "buffalo{}".format(platform.exe_suffix)
    return _download_executable(rctx, url, output)

def _provision_protoc(rctx, platform):
    url = _env(rctx, _TOOL_URL_ENVS["protoc"])
    if not url:
        version = _env(rctx, _VERSION_ENVS["protoc"], _DEFAULT_PROTOC_VERSION)
        url = _protoc_url(version, platform)
    # Archive layout: bin/protoc(.exe), include/google/protobuf/...
    bin_rel = "bin/protoc{}".format(platform.exe_suffix)
    return _download_and_extract_tool(rctx, url, "_protoc", bin_rel)

def _provision_protoc_gen_go(rctx, platform):
    url = _env(rctx, _TOOL_URL_ENVS["protoc-gen-go"])
    if not url:
        version = _env(rctx, _VERSION_ENVS["protoc-gen-go"], _DEFAULT_PROTOC_GEN_GO_VERSION)
        url = _protoc_gen_go_url(version, platform)
    bin_rel = "protoc-gen-go{}".format(platform.exe_suffix)
    return _download_and_extract_tool(rctx, url, "_protoc_gen_go", bin_rel)

def _provision_protoc_gen_go_grpc(rctx, platform):
    url = _env(rctx, _TOOL_URL_ENVS["protoc-gen-go-grpc"])
    if not url:
        version = _env(rctx, _VERSION_ENVS["protoc-gen-go-grpc"], _DEFAULT_PROTOC_GEN_GO_GRPC_VERSION)
        url = _protoc_gen_go_grpc_url(version, platform)
    bin_rel = "protoc-gen-go-grpc{}".format(platform.exe_suffix)
    return _download_and_extract_tool(rctx, url, "_protoc_gen_go_grpc", bin_rel)

# ---------------------------------------------------------------------------
# Compatibility fallbacks for tools that are not yet hermetic.
# These will be eliminated in subsequent iterations (Python via rules_python,
# Rust via rules_rust).
# ---------------------------------------------------------------------------

def _python_candidates(rctx, is_windows):
    candidates = []
    python = rctx.which("python")
    if python:
        candidates.append([str(python)])
    python3 = rctx.which("python3")
    if python3 and (not python or str(python3) != str(python)):
        candidates.append([str(python3)])
    if is_windows:
        py_launcher = rctx.which("py")
        if py_launcher:
            candidates.append([str(py_launcher), "-3"])
    return candidates

def _bootstrap_grpc_python_shim(rctx, platform):
    """Install grpcio-tools via host pip and emit a protoc-gen-grpc_python shim.

    This is the only remaining host-Python dependency; it will be removed
    when the rules_python integration lands.
    """
    candidates = _python_candidates(rctx, platform.is_windows)
    if not candidates:
        return None

    site_packages = str(rctx.path("_buffalo_python_site_packages"))
    last_err = ""
    selected = None

    for base in candidates:
        install = list(base) + [
            "-m", "pip", "install",
            "--disable-pip-version-check", "--no-input",
            "--target", site_packages,
            "grpcio-tools", "protobuf",
        ]
        res = rctx.execute(install)
        if res.return_code != 0:
            # Try ensurepip then retry once.
            rctx.execute(list(base) + ["-m", "ensurepip", "--upgrade"])
            res = rctx.execute(install)
        if res.return_code == 0:
            probe = rctx.execute(
                list(base) + ["-c", "import grpc_tools.protoc"],
                environment = {"PYTHONPATH": site_packages},
            )
            if probe.return_code == 0:
                selected = base
                break
        last_err = "{}\n{}".format(" ".join(base), res.stderr)

    if not selected:
        fail("Failed to bootstrap grpcio-tools for Python gRPC plugin.\n{}".format(last_err))

    shim_name = "protoc-gen-grpc_python.bat" if platform.is_windows else "protoc-gen-grpc_python"
    py_exe = selected[0]
    extra = " ".join(selected[1:])
    if platform.is_windows:
        pp = site_packages.replace("/", "\\")
        content = (
            "@echo off\r\n"
            "set \"PYTHONPATH={pp};%PYTHONPATH%\"\r\n"
            "\"{py}\" {extra} -m grpc_tools.protoc --grpc_python_out=. %*\r\n"
        ).format(pp = pp, py = py_exe, extra = extra)
    else:
        content = (
            "#!/usr/bin/env sh\n"
            "export PYTHONPATH='{pp}':\"${{PYTHONPATH:-}}\"\n"
            "exec \"{py}\" {extra} -m grpc_tools.protoc --grpc_python_out=. \"$@\"\n"
        ).format(pp = site_packages.replace("'", "'\\''"), py = py_exe, extra = extra)

    rctx.file(shim_name, content, executable = True)
    return rctx.path(shim_name)

def _bootstrap_rust_plugins(rctx, platform):
    """Install protoc-gen-prost / protoc-gen-tonic via host cargo.

    Returns (prost_path | None, tonic_path | None). Will be replaced by
    a rules_rust-based hermetic implementation.
    """
    cargo = rctx.which("cargo")
    if not cargo:
        return (None, None)

    install_root = str(rctx.path("_buffalo_rust_bin"))
    env = {
        "CARGO_INSTALL_ROOT": install_root,
        "CARGO_HOME": str(rctx.path("_buffalo_cargo_home")),
        "RUSTUP_HOME": str(rctx.path("_buffalo_rustup_home")),
    }

    prost_path = None
    tonic_path = None

    res = rctx.execute([str(cargo), "install", "protoc-gen-prost"], environment = env)
    if res.return_code == 0:
        candidate = rctx.path("_buffalo_rust_bin/bin/protoc-gen-prost{}".format(platform.exe_suffix))
        if candidate.exists:
            prost_path = candidate

    res = rctx.execute([str(cargo), "install", "protoc-gen-tonic"], environment = env)
    if res.return_code == 0:
        candidate = rctx.path("_buffalo_rust_bin/bin/protoc-gen-tonic{}".format(platform.exe_suffix))
        if candidate.exists:
            tonic_path = candidate

    return (prost_path, tonic_path)

# ---------------------------------------------------------------------------
# Optional URL overrides (single-file downloads, used by strict mode and
# by users who want to pin to internal mirrors).
# ---------------------------------------------------------------------------

def _maybe_download_from_env(rctx, tool_name, suffix):
    env_name = _TOOL_URL_ENVS.get(tool_name)
    if not env_name:
        return None
    url = _env(rctx, env_name)
    if not url:
        return None
    return _download_executable(rctx, url, "{}{}".format(tool_name, suffix))

# ---------------------------------------------------------------------------
# Repository rule implementation.
# ---------------------------------------------------------------------------

def _buffalo_toolchain_repo_impl(rctx):
    platform = _detect_platform(rctx)
    suffix = platform.exe_suffix
    strict = _strict_mode(rctx)

    # Hermetic upstream tools (always download; env URL overrides if set).
    buffalo = _maybe_download_from_env(rctx, "buffalo", suffix) or _provision_buffalo(rctx, platform)
    protoc = _maybe_download_from_env(rctx, "protoc", suffix) or _provision_protoc(rctx, platform)
    protoc_gen_go = _maybe_download_from_env(rctx, "protoc-gen-go", suffix) or _provision_protoc_gen_go(rctx, platform)
    protoc_gen_go_grpc = _maybe_download_from_env(rctx, "protoc-gen-go-grpc", suffix) or _provision_protoc_gen_go_grpc(rctx, platform)

    # Optional plugins: not yet hermetic. Allow strict mode to require URL.
    protoc_gen_grpc_python = _maybe_download_from_env(rctx, "protoc-gen-grpc_python", suffix)
    protoc_gen_prost = _maybe_download_from_env(rctx, "protoc-gen-prost", suffix)
    protoc_gen_tonic = _maybe_download_from_env(rctx, "protoc-gen-tonic", suffix)

    if strict:
        missing = []
        if not protoc_gen_grpc_python:
            missing.append(_TOOL_URL_ENVS["protoc-gen-grpc_python"])
        if missing:
            fail("Strict sandbox mode ({}=1) requires URLs for: {}".format(
                _STRICT_MODE_ENV,
                ", ".join(missing),
            ))
    else:
        if not protoc_gen_grpc_python:
            protoc_gen_grpc_python = _bootstrap_grpc_python_shim(rctx, platform)
        if not protoc_gen_prost or not protoc_gen_tonic:
            prost, tonic = _bootstrap_rust_plugins(rctx, platform)
            protoc_gen_prost = protoc_gen_prost or prost
            protoc_gen_tonic = protoc_gen_tonic or tonic

    # ---- Stage every binary under a stable, predictable filename. -------
    # Keep filename of the python shim if it's a .bat, so Windows executes
    # it correctly.
    grpc_python_target = None
    if protoc_gen_grpc_python:
        src_name = str(protoc_gen_grpc_python).replace("\\", "/").split("/")[-1]
        if src_name.endswith(".bat"):
            grpc_python_target = "protoc-gen-grpc_python.bat"
        else:
            grpc_python_target = "protoc-gen-grpc_python{}".format(suffix)

    files = {
        "buffalo{}".format(suffix): buffalo,
        "protoc{}".format(suffix): protoc,
        "protoc-gen-go{}".format(suffix): protoc_gen_go,
        "protoc-gen-go-grpc{}".format(suffix): protoc_gen_go_grpc,
    }
    if protoc_gen_grpc_python:
        files[grpc_python_target] = protoc_gen_grpc_python
    if protoc_gen_prost:
        files["protoc-gen-prost{}".format(suffix)] = protoc_gen_prost
    if protoc_gen_tonic:
        files["protoc-gen-tonic{}".format(suffix)] = protoc_gen_tonic

    for target_name, source in files.items():
        # Skip self-symlink if download_executable already produced the file
        # at the target name.
        if str(source).replace("\\", "/").endswith("/" + target_name):
            continue
        rctx.symlink(source, target_name)

    # Compatibility stub for protoc-gen-grpc_python in strict mode without URL.
    if not protoc_gen_grpc_python:
        stub_name = "protoc-gen-grpc_python{}".format(suffix)
        if platform.is_windows:
            rctx.file(
                stub_name,
                "@echo off\r\necho protoc-gen-grpc_python is not provisioned.\r\nexit /b 1\r\n",
                executable = True,
            )
        else:
            rctx.file(
                stub_name,
                "#!/usr/bin/env sh\necho 'protoc-gen-grpc_python is not provisioned.' >&2\nexit 1\n",
                executable = True,
            )
        grpc_python_target = stub_name

    # ---- Generate BUILD.bazel ------------------------------------------
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
    if protoc_gen_prost:
        name = "protoc-gen-prost{}".format(suffix)
        exports.append(name)
        aliases.append(("protoc_gen_prost_bin", name))
    if protoc_gen_tonic:
        name = "protoc-gen-tonic{}".format(suffix)
        exports.append(name)
        aliases.append(("protoc_gen_tonic_bin", name))

    exports_block = ",\n".join(['    "{}"'.format(n) for n in exports])
    aliases_block = "\n\n".join([
        'alias(\n    name = "{}",\n    actual = ":{}",\n)'.format(alias_name, target)
        for alias_name, target in aliases
    ])

    rctx.file("BUILD.bazel", content = """package(default_visibility = ["//visibility:public"])

exports_files([
{exports}
])

{aliases}
""".format(exports = exports_block, aliases = aliases_block))

# ---------------------------------------------------------------------------
# Public repository rule.
# ---------------------------------------------------------------------------

buffalo_toolchain_repo = repository_rule(
    implementation = _buffalo_toolchain_repo_impl,
    environ = [
        "PATH",
        "HOME",
        "USERPROFILE",
        "TMP",
        "TEMP",
        _STRICT_MODE_ENV,
        _BUFFALO_REPO_ENV,
        _VERSION_ENVS["protoc"],
        _VERSION_ENVS["protoc-gen-go"],
        _VERSION_ENVS["protoc-gen-go-grpc"],
        _VERSION_ENVS["buffalo"],
        _TOOL_URL_ENVS["buffalo"],
        _TOOL_URL_ENVS["protoc"],
        _TOOL_URL_ENVS["protoc-gen-go"],
        _TOOL_URL_ENVS["protoc-gen-go-grpc"],
        _TOOL_URL_ENVS["protoc-gen-grpc_python"],
        _TOOL_URL_ENVS["protoc-gen-prost"],
        _TOOL_URL_ENVS["protoc-gen-tonic"],
    ],
    doc = "Hermetically provisions Buffalo CLI and protoc tooling by downloading pinned upstream releases. Falls back to host Python/cargo only for plugins that are not yet hermetic (grpc_python, prost, tonic).",
)
