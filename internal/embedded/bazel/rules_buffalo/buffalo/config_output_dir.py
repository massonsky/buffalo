"""Print normalized Buffalo output.base_dir from a config file."""

from __future__ import annotations

import sys


DEFAULT_OUTPUT = "gen"


def _normalize(value: str) -> str:
    value = value.strip().strip('"\'')
    value = value.replace("\\", "/")
    while value.startswith("./"):
        value = value[2:]
    value = value.strip("/")
    return value or DEFAULT_OUTPUT


def _parse_without_yaml(config_path: str) -> str:
    with open(config_path, "r", encoding="utf-8") as fh:
        in_output_section = False
        output_indent = 0
        for raw_line in fh:
            line = raw_line.split("#", 1)[0].rstrip("\r\n")
            stripped = line.strip()
            if not stripped:
                continue

            indent = len(line) - len(line.lstrip(" "))
            if not in_output_section:
                if stripped == "output:":
                    in_output_section = True
                    output_indent = indent
                continue

            if indent <= output_indent:
                break

            if stripped.startswith("base_dir:"):
                return _normalize(stripped.split(":", 1)[1])

    return DEFAULT_OUTPUT


def read_output_dir(config_path: str) -> str:
    try:
        import yaml  # type: ignore
    except Exception:
        yaml = None

    if yaml is not None:
        with open(config_path, "r", encoding="utf-8") as fh:
            data = yaml.safe_load(fh) or {}
        output = data.get("output") or {}
        if isinstance(output, dict) and output.get("base_dir"):
            return _normalize(str(output["base_dir"]))

    return _parse_without_yaml(config_path)


def main() -> int:
    if len(sys.argv) != 2:
        print(DEFAULT_OUTPUT)
        return 0

    print(read_output_dir(sys.argv[1]))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())