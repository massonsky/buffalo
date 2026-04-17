"""Module extensions for rules_buffalo (bzlmod)."""

load(":repositories.bzl", "buffalo_toolchain_repo")

def _buffalo_impl(module_ctx):
    """Registers the buffalo toolchain repository."""
    buffalo_toolchain_repo(name = "buffalo_toolchain")

buffalo = module_extension(
    implementation = _buffalo_impl,
    doc = "Module extension that locates the Buffalo binary and creates a toolchain repository.",
)
