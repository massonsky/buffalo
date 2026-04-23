"""Repository rules for locating the Buffalo binary."""

def _buffalo_toolchain_repo_impl(rctx):
    buffalo = rctx.which("buffalo")
    if not buffalo:
        fail(
            "Buffalo binary not found in PATH.\n" +
            "Install: go install github.com/massonsky/buffalo/cmd/buffalo@latest",
        )

    is_windows = rctx.os.name.lower().startswith("windows")
    bin_name = "buffalo.exe" if is_windows else "buffalo"

    rctx.symlink(buffalo, bin_name)
    rctx.file("BUILD.bazel", content = """\
package(default_visibility = ["//visibility:public"])

exports_files(["{bin_name}"])

alias(
    name = "buffalo_bin",
    actual = ":{bin_name}",
)
""".format(bin_name = bin_name))

buffalo_toolchain_repo = repository_rule(
    implementation = _buffalo_toolchain_repo_impl,
    environ = ["PATH"],
    doc = "Locates the buffalo binary on the system PATH and creates a repository with it.",
)
