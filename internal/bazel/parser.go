package bazel

import (
	"os"
	"regexp"
	"strings"
)

// ParseBuildFile parses a BUILD/BUILD.bazel file and extracts proto_library targets.
func ParseBuildFile(path string, pkg string) ([]BazelTarget, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseBuildContent(string(data), pkg), nil
}

// parseBuildContent extracts targets from BUILD file content.
// This is a pragmatic parser — it handles the common patterns used in proto_library,
// go_proto_library, py_proto_library, etc. without a full Starlark evaluator.
func parseBuildContent(content string, pkg string) []BazelTarget {
	var targets []BazelTarget

	// Match rule invocations: rule_name(\n    ...\n)
	rulePattern := regexp.MustCompile(`(?m)^(\w+)\(\s*\n((?:.*\n)*?)\)`)
	matches := rulePattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		rule := match[1]
		body := match[2]

		target := BazelTarget{
			Rule:    rule,
			Package: pkg,
			Extra:   make(map[string]string),
		}

		target.Name = parseStringAttr(body, "name")
		target.Srcs = parseListAttr(body, "srcs")
		target.Deps = parseListAttr(body, "deps")
		target.Visibility = parseListAttr(body, "visibility")
		target.Tags = parseListAttr(body, "tags")
		target.StripImportPrefix = parseStringAttr(body, "strip_import_prefix")
		target.ImportPrefix = parseStringAttr(body, "import_prefix")
		target.ProtoSourceRoot = parseStringAttr(body, "proto_source_root")

		if target.Name != "" {
			targets = append(targets, target)
		}
	}

	return targets
}

// parseStringAttr extracts a string attribute value: name = "value",
func parseStringAttr(body, attr string) string {
	pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(attr) + `\s*=\s*"([^"]*)"`)
	m := pattern.FindStringSubmatch(body)
	if m != nil {
		return m[1]
	}
	return ""
}

// parseListAttr extracts a list attribute: srcs = ["a.proto", "b.proto"],
// Handles multi-line lists and glob() calls.
func parseListAttr(body, attr string) []string {
	// Try inline list: attr = ["a", "b"]
	inlinePattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(attr) + `\s*=\s*\[([^\]]*)\]`)
	m := inlinePattern.FindStringSubmatch(body)
	if m != nil {
		return extractQuotedStrings(m[1])
	}

	// Try multi-line list
	multiPattern := regexp.MustCompile(`(?ms)^\s*` + regexp.QuoteMeta(attr) + `\s*=\s*\[\s*\n(.*?)\]`)
	m = multiPattern.FindStringSubmatch(body)
	if m != nil {
		return extractQuotedStrings(m[1])
	}

	// Try glob() call: srcs = glob(["*.proto"])
	globPattern := regexp.MustCompile(`(?ms)^\s*` + regexp.QuoteMeta(attr) + `\s*=\s*glob\(\s*\[([^\]]*)\]\s*\)`)
	m = globPattern.FindStringSubmatch(body)
	if m != nil {
		return extractQuotedStrings(m[1])
	}

	return nil
}

// extractQuotedStrings pulls all double-quoted strings from a comma-separated text.
func extractQuotedStrings(text string) []string {
	pattern := regexp.MustCompile(`"([^"]*)"`)
	matches := pattern.FindAllStringSubmatch(text, -1)
	var result []string
	for _, m := range matches {
		result = append(result, m[1])
	}
	return result
}

// parseModuleName extracts module(name = "...") from MODULE.bazel content.
func parseModuleName(content string) string {
	pattern := regexp.MustCompile(`(?ms)module\(\s*\n(?:.*\n)*?\s*name\s*=\s*"([^"]*)"`)
	m := pattern.FindStringSubmatch(content)
	if m != nil {
		return m[1]
	}
	// Try inline: module(name = "foo", ...)
	pattern2 := regexp.MustCompile(`module\(\s*name\s*=\s*"([^"]*)"`)
	m = pattern2.FindStringSubmatch(content)
	if m != nil {
		return m[1]
	}
	return ""
}

// FilterProtoTargets returns only proto_library targets from a list.
func FilterProtoTargets(targets []BazelTarget) []BazelTarget {
	var result []BazelTarget
	for _, t := range targets {
		if t.Rule == "proto_library" {
			result = append(result, t)
		}
	}
	return result
}

// FilterLangProtoTargets returns language-specific proto library targets.
func FilterLangProtoTargets(targets []BazelTarget, lang string) []BazelTarget {
	ruleNames := langProtoRules(lang)
	var result []BazelTarget
	for _, t := range targets {
		for _, r := range ruleNames {
			if t.Rule == r {
				result = append(result, t)
				break
			}
		}
	}
	return result
}

// langProtoRules returns the known Bazel rule names for language-specific proto libraries.
func langProtoRules(lang string) []string {
	switch lang {
	case "go":
		return []string{"go_proto_library", "go_grpc_library"}
	case "python":
		return []string{"py_proto_library", "py_grpc_library"}
	case "cpp":
		return []string{"cc_proto_library", "cc_grpc_library"}
	case "rust":
		return []string{"rust_proto_library", "rust_grpc_library"}
	case "typescript":
		return []string{"ts_proto_library"}
	default:
		return nil
	}
}

// ResolveProtoFiles resolves the actual file paths for a target's srcs,
// relative to the workspace root and the target's package directory.
func ResolveProtoFiles(ws *Workspace, target BazelTarget) []string {
	// Determine the package directory on disk
	pkgDir := target.Package
	pkgDir = strings.TrimPrefix(pkgDir, "//")

	var files []string
	for _, src := range target.Srcs {
		// Handle label references (e.g., "//other:file.proto")
		if strings.HasPrefix(src, "//") || strings.HasPrefix(src, ":") || strings.Contains(src, ":") {
			// Skip label references — they should be resolved via deps
			continue
		}
		fullPath := src
		if pkgDir != "" {
			fullPath = pkgDir + "/" + src
		}
		files = append(files, fullPath)
	}
	return files
}
