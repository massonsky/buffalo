"""Cross-platform runner for buffalo_proto_gen.

Invoked via `bazel run //<target>`. Generates code into the source tree
using BUILD_WORKSPACE_DIRECTORY (set by `bazel run`).
"""
import argparse
import os
import re
import shutil
import subprocess
import sys
import tempfile


def _normalize(path: str) -> str:
    value = (path or "generated").strip().strip('"').strip("'")
    if not value:
        value = "generated"
    value = value.replace("\\", "/")
    if value.startswith("./"):
        value = value[2:]
    value = value.strip("/")
    return value or "generated"


def _read_output_dir(config_path: str) -> str:
    try:
        import yaml  # type: ignore

        with open(config_path, "r", encoding="utf-8") as fh:
            data = yaml.safe_load(fh) or {}
        return _normalize(((data.get("output") or {}).get("base_dir")))
    except Exception:
        pass

    # fallback without PyYAML: simple, robust extraction for `output.base_dir`
    try:
        with open(config_path, "r", encoding="utf-8") as fh:
            lines = fh.readlines()
    except Exception:
        return "generated"

    in_output = False
    output_indent = None
    for line in lines:
        raw = line.rstrip("\n")
        if not raw.strip() or raw.lstrip().startswith("#"):
            continue

        indent = len(raw) - len(raw.lstrip(" "))
        text = raw.strip()

        if re.match(r"^output\s*:\s*$", text):
            in_output = True
            output_indent = indent
            continue

        if in_output and output_indent is not None and indent <= output_indent:
            in_output = False
            output_indent = None

        if in_output and text.startswith("base_dir"):
            _, _, value = text.partition(":")
            return _normalize(value)

    return "generated"


def _copy_new_files(src_dir: str, dst_dir: str) -> tuple[int, int]:
    copied = 0
    skipped = 0
    os.makedirs(dst_dir, exist_ok=True)

    for current_root, dirnames, filenames in os.walk(src_dir):
        rel_root = os.path.relpath(current_root, src_dir)
        dst_root = dst_dir if rel_root == "." else os.path.join(dst_dir, rel_root)
        os.makedirs(dst_root, exist_ok=True)

        for dirname in dirnames:
            os.makedirs(os.path.join(dst_root, dirname), exist_ok=True)

        for filename in filenames:
            src_file = os.path.join(current_root, filename)
            dst_file = os.path.join(dst_root, filename)
            if os.path.exists(dst_file):
                skipped += 1
                continue
            shutil.copy2(src_file, dst_file)
            copied += 1

    return copied, skipped


def _run_bazel_build(root: str, target: str) -> None:
    if not target:
        return
    cmd = ["bazel", "build", target]
    print("buffalo: build =", " ".join(cmd))
    result = subprocess.run(cmd, cwd=root)
    if result.returncode != 0:
        sys.exit(result.returncode)


def _prepend_grpc_python_plugin_dir() -> None:
    try:
        import importlib.resources

        plugin_path = str(importlib.resources.files("grpc_tools").joinpath("grpc_python_plugin"))
        plugin_dir = os.path.dirname(plugin_path)
        if plugin_dir:
            os.environ["PATH"] = plugin_dir + os.pathsep + os.environ.get("PATH", "")
            print("buffalo: grpc_python_plugin dir =", plugin_dir)
    except Exception as exc:
        print("buffalo: warning: could not locate grpc_python_plugin:", exc)


def _find_grpc_tools_python():
    candidates = []
    if os.name == "nt":
        candidates.append(["py", "-3"])
    candidates.append(["python"])
    candidates.append(["python3"])

    for cmd in candidates:
        result = subprocess.run(
            cmd + ["-c", "import grpc_tools.protoc"],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
        if result.returncode == 0:
            return cmd
    return None


def _prepend_protoc_shim_dir():
    python_cmd = _find_grpc_tools_python()
    if not python_cmd:
        return None

    tool_dir = tempfile.mkdtemp(prefix="buffalo_tools_")
    if os.name == "nt":
        shim_path = os.path.join(tool_dir, "protoc.bat")
        extra = " ".join(python_cmd[1:])
        with open(shim_path, "w", encoding="utf-8", newline="\r\n") as fh:
            fh.write("@echo off\n")
            fh.write('"{}" {} -m grpc_tools.protoc %*\n'.format(python_cmd[0], extra).strip() + "\n")
    else:
        shim_path = os.path.join(tool_dir, "protoc")
        extra = " ".join(python_cmd[1:])
        with open(shim_path, "w", encoding="utf-8", newline="\n") as fh:
            fh.write("#!/usr/bin/env sh\n")
            fh.write('exec "{}" {} -m grpc_tools.protoc "$@"\n'.format(python_cmd[0], extra).strip() + "\n")
        os.chmod(shim_path, 0o755)

    os.environ["PATH"] = tool_dir + os.pathsep + os.environ.get("PATH", "")
    print("buffalo: protoc shim dir =", tool_dir)
    return tool_dir


def main():
    root = os.environ.get("BUILD_WORKSPACE_DIRECTORY", os.getcwd())
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--config", default="buffalo.yaml")
    parser.add_argument("--copy-from-bazel-bin", default="")
    parser.add_argument("--build-target", default="")
    known, _ = parser.parse_known_args(sys.argv[1:])

    if known.copy_from_bazel_bin:
        config_path = known.config
        if not os.path.isabs(config_path):
            config_path = os.path.join(root, config_path)
        dst_rel = _read_output_dir(config_path)
        src_dir = os.path.join(root, "bazel-bin", known.copy_from_bazel_bin)
        dst_dir = os.path.normpath(os.path.join(root, dst_rel))

        print("buffalo: copy mode")
        print("buffalo: source =", src_dir)
        print("buffalo: target =", dst_dir)

        _run_bazel_build(root, known.build_target)

        if not os.path.isdir(src_dir):
            print("buffalo: error: source directory not found:", src_dir)
            print("buffalo: hint: build compile target first (e.g. //:buffalo_compile)")
            sys.exit(1)

        copied, skipped = _copy_new_files(src_dir, dst_dir)
        print(
            "buffalo: copied generated files from bazel-bin to config output dir "
            f"(copied={copied}, skipped_existing={skipped})"
        )
        sys.exit(0)

    _prepend_protoc_shim_dir()
    _prepend_grpc_python_plugin_dir()
    cmd = ["buffalo", "build", "--skip-system-check", "--skip-lock"]
    # sys.argv[1:] contains args passed via bazel run target's `args` attr
    # plus any additional args after `--`
    cmd.extend(sys.argv[1:])
    print("buffalo: cwd =", root)
    print("buffalo:", " ".join(cmd))
    result = subprocess.run(cmd, cwd=root)
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
