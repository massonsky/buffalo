"""Cross-platform runner for buffalo_proto_gen.

Invoked via `bazel run //<target>`. Generates code into the source tree
using BUILD_WORKSPACE_DIRECTORY (set by `bazel run`).
"""
import os
import subprocess
import sys


def main():
    root = os.environ.get("BUILD_WORKSPACE_DIRECTORY", os.getcwd())
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
