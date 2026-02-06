package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ──────────────────────────────────────────────────────────────────
// Annotation regex patterns
// ──────────────────────────────────────────────────────────────────

// annotationRe matches a single buffalo.validate annotation within field options.
//
// Examples it captures:
//
//	[(buffalo.validate.rules).double  = {gte: -90, lte: 90}]
//	[(buffalo.validate.rules).string  = {email: true}]
//	[(buffalo.validate.rules).required = true]
var annotationRe = regexp.MustCompile(
	`\(\s*buffalo\.validate(?:\.rules)?\s*\)\s*\.(\w+)\s*=\s*(\{[^}]*\}|[^\],\s]+)`,
)

// kvRe matches key: value pairs inside { }.
var kvRe = regexp.MustCompile(`(\w+)\s*:\s*("(?:[^"\\]|\\.)*"|[^,}\s]+)`)

// ──────────────────────────────────────────────────────────────────
// Public API
// ──────────────────────────────────────────────────────────────────

// ParseFieldAnnotation parses all buffalo.validate annotations from a field's
// option string and returns the list of parsed rules.
//
// annotation is the full [...] option portion of a proto field line.
// fieldName / fieldType are used to tag each returned FieldRule.
func ParseFieldAnnotation(annotation, fieldName, fieldType string) ([]FieldRule, error) {
	matches := annotationRe.FindAllStringSubmatch(annotation, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var rules []FieldRule
	for _, m := range matches {
		category := m[1] // e.g. "double", "string", "required"
		body := strings.TrimSpace(m[2])

		parsed, err := parseCategory(category, body, fieldName, fieldType)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", fieldName, err)
		}
		rules = append(rules, parsed...)
	}
	return rules, nil
}

// ExtractValidationRules scans a full proto file text and returns
// MessageRules for every message that has at least one buffalo.validate
// annotation.
func ExtractValidationRules(content, filePath string) ([]MessageRules, error) {
	pkg := extractPackage(content)
	messages := extractMessages(content)

	var results []MessageRules
	for _, msg := range messages {
		mr := MessageRules{
			MessageName: msg.name,
			Package:     pkg,
			FilePath:    filePath,
			Disabled:    msg.disabled,
			Fields:      make(map[string][]FieldRule),
		}

		if msg.disabled {
			results = append(results, mr)
			continue
		}

		for _, f := range msg.fields {
			if !strings.Contains(f.options, "buffalo.validate") {
				continue
			}
			rules, err := ParseFieldAnnotation(f.options, f.name, f.typ)
			if err != nil {
				return nil, fmt.Errorf("%s: message %s: %w", filePath, msg.name, err)
			}
			if len(rules) > 0 {
				mr.Fields[f.name] = rules
			}
		}

		if len(mr.Fields) > 0 {
			results = append(results, mr)
		}
	}
	return results, nil
}

// ──────────────────────────────────────────────────────────────────
// Internal: parse a single annotation category
// ──────────────────────────────────────────────────────────────────

func parseCategory(category, body, fieldName, fieldType string) ([]FieldRule, error) {
	// Bare boolean rules: [(buffalo.validate.rules).required = true]
	switch category {
	case "required":
		if body == "true" {
			return []FieldRule{{
				Type:      RuleRequired,
				Value:     true,
				FieldName: fieldName,
				FieldType: fieldType,
			}}, nil
		}
		return nil, nil
	case "message":
		// custom error message, not a rule itself — attach via Message field
		msg := strings.Trim(body, `"' `)
		return []FieldRule{{
			Type:      "message",
			Value:     msg,
			FieldName: fieldName,
			FieldType: fieldType,
			Message:   msg,
		}}, nil
	}

	// Structured rules: { key: value, ... }
	if !(strings.HasPrefix(body, "{") && strings.HasSuffix(body, "}")) {
		return nil, fmt.Errorf("invalid rule body for '%s': expected { ... }, got %q", category, body)
	}

	inner := body[1 : len(body)-1]
	kvs := kvRe.FindAllStringSubmatch(inner, -1)

	var rules []FieldRule
	for _, kv := range kvs {
		key := kv[1]
		rawValue := strings.Trim(kv[2], `"' `)

		rule, err := buildRule(key, rawValue, fieldName, fieldType)
		if err != nil {
			return nil, fmt.Errorf("rule '%s.%s': %w", category, key, err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// ──────────────────────────────────────────────────────────────────
// Internal: build a single FieldRule from key=value
// ──────────────────────────────────────────────────────────────────

func buildRule(key, rawValue, fieldName, fieldType string) (FieldRule, error) {
	r := FieldRule{FieldName: fieldName, FieldType: fieldType}

	switch key {
	// ── Numeric comparisons ──
	case "gt":
		v, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return r, fmt.Errorf("gt must be numeric: %w", err)
		}
		r.Type, r.Value = RuleGt, v
	case "gte":
		v, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return r, fmt.Errorf("gte must be numeric: %w", err)
		}
		r.Type, r.Value = RuleGte, v
	case "lt":
		v, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return r, fmt.Errorf("lt must be numeric: %w", err)
		}
		r.Type, r.Value = RuleLt, v
	case "lte":
		v, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return r, fmt.Errorf("lte must be numeric: %w", err)
		}
		r.Type, r.Value = RuleLte, v
	case "const":
		v, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return r, fmt.Errorf("const must be numeric: %w", err)
		}
		r.Type, r.Value = RuleConst, v

	// ── String length ──
	case "min_len":
		v, err := strconv.ParseUint(rawValue, 10, 64)
		if err != nil {
			return r, fmt.Errorf("min_len must be unsigned integer: %w", err)
		}
		r.Type, r.Value = RuleMinLen, v
	case "max_len":
		v, err := strconv.ParseUint(rawValue, 10, 64)
		if err != nil {
			return r, fmt.Errorf("max_len must be unsigned integer: %w", err)
		}
		r.Type, r.Value = RuleMaxLen, v

	// ── String format ──
	case "pattern":
		if _, err := regexp.Compile(rawValue); err != nil {
			return r, fmt.Errorf("invalid regex pattern %q: %w", rawValue, err)
		}
		r.Type, r.Value = RulePattern, rawValue
	case "prefix":
		r.Type, r.Value = RulePrefix, rawValue
	case "suffix":
		r.Type, r.Value = RuleSuffix, rawValue
	case "contains":
		r.Type, r.Value = RuleContains, rawValue

	// ── Boolean flags ──
	case "email":
		r.Type, r.Value = RuleEmail, rawValue == "true"
	case "uri":
		r.Type, r.Value = RuleURI, rawValue == "true"
	case "uuid":
		r.Type, r.Value = RuleUUID, rawValue == "true"
	case "not_empty":
		r.Type, r.Value = RuleNotEmpty, rawValue == "true"
	case "ip":
		r.Type, r.Value = RuleIP, rawValue == "true"
	case "ipv4":
		r.Type, r.Value = RuleIPv4, rawValue == "true"
	case "ipv6":
		r.Type, r.Value = RuleIPv6, rawValue == "true"
	case "hostname":
		r.Type, r.Value = RuleHostname, rawValue == "true"
	case "unique":
		r.Type, r.Value = RuleUnique, rawValue == "true"
	case "defined_only":
		r.Type, r.Value = "defined_only", rawValue == "true"

	// ── Collection sizes ──
	case "min_items":
		v, err := strconv.ParseUint(rawValue, 10, 64)
		if err != nil {
			return r, fmt.Errorf("min_items must be unsigned integer: %w", err)
		}
		r.Type, r.Value = RuleMinItems, v
	case "max_items":
		v, err := strconv.ParseUint(rawValue, 10, 64)
		if err != nil {
			return r, fmt.Errorf("max_items must be unsigned integer: %w", err)
		}
		r.Type, r.Value = RuleMaxItems, v
	case "min_pairs":
		v, err := strconv.ParseUint(rawValue, 10, 64)
		if err != nil {
			return r, fmt.Errorf("min_pairs must be unsigned integer: %w", err)
		}
		r.Type, r.Value = RuleMinPairs, v
	case "max_pairs":
		v, err := strconv.ParseUint(rawValue, 10, 64)
		if err != nil {
			return r, fmt.Errorf("max_pairs must be unsigned integer: %w", err)
		}
		r.Type, r.Value = RuleMaxPairs, v

	// ── Timestamp rules ──
	case "gt_now":
		r.Type, r.Value = RuleGtNow, rawValue == "true"
	case "lt_now":
		r.Type, r.Value = RuleLtNow, rawValue == "true"

	default:
		return r, fmt.Errorf("unknown validation rule: '%s'", key)
	}

	return r, nil
}

// ──────────────────────────────────────────────────────────────────
// Internal: lightweight proto text scanning
//
// We intentionally avoid a full protobuf parser: Buffalo already has
// its own parser in builder/parser.go, and here we only need to
// extract annotations. This keeps external dependencies at zero.
// ──────────────────────────────────────────────────────────────────

type rawMessage struct {
	name     string
	fields   []rawField
	disabled bool
}

type rawField struct {
	name    string
	typ     string
	options string // everything inside [...] after the field number
}

func extractPackage(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			pkg := strings.TrimPrefix(line, "package ")
			pkg = strings.TrimSuffix(strings.TrimSpace(pkg), ";")
			return strings.TrimSpace(pkg)
		}
	}
	return ""
}

func extractMessages(content string) []rawMessage {
	var messages []rawMessage
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); {
		line := strings.TrimSpace(lines[i])

		if !strings.HasPrefix(line, "message ") || !strings.Contains(line, "{") {
			i++
			continue
		}

		name := strings.TrimSpace(strings.TrimPrefix(line, "message "))
		name = strings.TrimSpace(strings.SplitN(name, "{", 2)[0])

		disabled := false
		// Check for a comment-based disable annotation on the same line
		if strings.Contains(line, "@buffalo.validate.disabled") {
			disabled = true
		}

		var fields []rawField
		braceDepth := 1
		i++

		for i < len(lines) && braceDepth > 0 {
			fline := lines[i]
			trimmed := strings.TrimSpace(fline)

			braceDepth += strings.Count(trimmed, "{")
			braceDepth -= strings.Count(trimmed, "}")

			// Only parse fields at depth == 1 (direct children)
			if braceDepth == 1 && strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "//") {
				if f := parseFieldLine(trimmed); f != nil {
					fields = append(fields, *f)
				}
			}

			i++
		}

		messages = append(messages, rawMessage{name: name, fields: fields, disabled: disabled})
	}

	return messages
}

func parseFieldLine(line string) *rawField {
	// Strip trailing comments that are NOT inside [...]
	bracketIdx := strings.Index(line, "[")
	if bracketIdx < 0 {
		// No options — strip all comments
		if ci := strings.Index(line, "//"); ci >= 0 {
			line = line[:ci]
		}
	} else {
		// There are options — only strip comments after the closing ]
		closingIdx := strings.LastIndex(line, "]")
		if closingIdx > bracketIdx {
			tail := line[closingIdx+1:]
			if ci := strings.Index(tail, "//"); ci >= 0 {
				line = line[:closingIdx+1+ci]
			}
		}
	}
	line = strings.TrimSpace(line)
	if line == "" || line == "}" || line == "{" {
		return nil
	}

	// Extract options portion [...] including nested parens
	options := ""
	if idx := strings.Index(line, "["); idx >= 0 {
		endIdx := strings.LastIndex(line, "]")
		if endIdx > idx {
			options = line[idx : endIdx+1]
		}
	}

	// Parse "type name = N" or "repeated type name = N"
	parts := strings.Fields(line)
	if len(parts) < 4 { // at least: type name = N
		return nil
	}

	offset := 0
	typ := parts[0]
	if typ == "repeated" || typ == "optional" || typ == "map" {
		if len(parts) < 5 {
			return nil
		}
		typ = parts[0] + " " + parts[1]
		offset = 1
	}
	name := parts[1+offset]

	// Verify there is an "=" sign
	if parts[2+offset] != "=" {
		return nil
	}

	return &rawField{
		name:    name,
		typ:     typ,
		options: options,
	}
}
