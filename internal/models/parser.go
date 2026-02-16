package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ──────────────────────────────────────────────────────────────────
// Annotation regex patterns
// ──────────────────────────────────────────────────────────────────

// modelAnnotationRe matches message-level buffalo.models.model annotations.
//
//	option (buffalo.models.model) = { ... };
var modelAnnotationRe = regexp.MustCompile(
	`option\s+\(\s*buffalo\.models\.model\s*\)\s*=\s*\{([^;]*)\}\s*;`,
)

// fieldAnnotationRe matches field-level buffalo.models.field annotations.
//
//	[(buffalo.models.field) = { ... }]
var fieldAnnotationRe = regexp.MustCompile(
	`\(\s*buffalo\.models\.field\s*\)\s*=\s*\{([^}]*)\}`,
)

// kvRe matches key: value pairs inside { }, including nested braces for relation.
var kvRe = regexp.MustCompile(`(\w+)\s*:\s*("(?:[^"\\]|\\.)*"|\{[^}]*\}|\[[^\]]*\]|[^,}\s]+)`)

// messageBlockRe matches message blocks.
var messageBlockRe = regexp.MustCompile(`(?m)^message\s+(\w+)\s*\{`)

// packageRe matches the package declaration.
var packageRe = regexp.MustCompile(`(?m)^\s*package\s+([\w.]+)\s*;`)

// fieldLineRe parses a proto field line.
// It captures the field definition without the annotation part.
// Multi-line annotations are handled separately.
//
//	repeated string tags = 5;
//	optional float speed = 6;
//	string name = 1 [(buffalo.models.field) = { ... }];
var fieldLineRe = regexp.MustCompile(
	`(?m)^\s*((?:repeated|optional)\s+)?(\w[\w.]*)\s+(\w+)\s*=\s*(\d+)`,
)

// listValueRe matches list values: ["a", "b", "c"]
var listValueRe = regexp.MustCompile(`"([^"]*)"`)

// mapFieldRe matches map<KeyType, ValueType> field_name = N;
var mapFieldRe = regexp.MustCompile(
	`(?m)^\s*map\s*<\s*(\w[\w.]*)\s*,\s*(\w[\w.]*)\s*>\s+(\w+)\s*=\s*(\d+)`,
)

// oneofBlockRe matches oneof blocks: oneof name { ... }
var oneofBlockRe = regexp.MustCompile(`(?m)^\s*oneof\s+(\w+)\s*\{`)

// enumBlockRe matches enum blocks: enum Name { ... }
var enumBlockRe = regexp.MustCompile(`(?m)^\s*enum\s+(\w+)\s*\{`)

// enumValueRe matches enum value lines: NAME = N;
var enumValueRe = regexp.MustCompile(`(?m)^\s*(\w+)\s*=\s*(-?\d+)\s*;`)

// commentRe matches single-line comments.
var commentRe = regexp.MustCompile(`(?m)^\s*//\s*(.*)$`)

// serviceBlockRe matches service blocks (to skip them).
var serviceBlockRe = regexp.MustCompile(`(?m)^service\s+(\w+)\s*\{`)

// ──────────────────────────────────────────────────────────────────
// Public API
// ──────────────────────────────────────────────────────────────────

// ExtractModels scans a full proto file text and returns ModelDefs
// for every message that has a [(buffalo.models.model)] annotation.
func ExtractModels(content, filePath string) ([]ModelDef, error) {
	pkg := extractPackage(content)
	messages := extractMessageBlocks(content)

	var results []ModelDef
	for _, msg := range messages {
		// Check if this message has a buffalo.models.model annotation
		modelMatch := modelAnnotationRe.FindStringSubmatch(msg.body)
		if modelMatch == nil {
			continue
		}

		md := ModelDef{
			MessageName: msg.name,
			Package:     pkg,
			FilePath:    filePath,
			Fields:      []FieldDef{},
		}

		// Parse model-level options
		if err := parseModelOptions(modelMatch[1], &md); err != nil {
			return nil, fmt.Errorf("%s: message %s: %w", filePath, msg.name, err)
		}

		// Parse field-level annotations
		fieldLines := fieldLineRe.FindAllStringSubmatchIndex(msg.body, -1)
		for _, flIdx := range fieldLines {
			fl := []string{
				msg.body[flIdx[0]:flIdx[1]], // full match
				"",                          // repeated (placeholder)
				"",                          // type
				"",                          // name
				"",                          // number
			}
			if flIdx[2] >= 0 {
				fl[1] = msg.body[flIdx[2]:flIdx[3]]
			}
			fl[2] = msg.body[flIdx[4]:flIdx[5]]
			fl[3] = msg.body[flIdx[6]:flIdx[7]]
			fl[4] = msg.body[flIdx[8]:flIdx[9]]

			qualifier := strings.TrimSpace(fl[1])
			fd := FieldDef{
				Repeated:  qualifier == "repeated",
				Nullable:  qualifier == "optional",
				ProtoType: fl[2],
				Name:      fl[3],
			}
			if n, err := strconv.Atoi(fl[4]); err == nil {
				fd.Number = n
			}

			// Find the annotation for this field: look from end of field def
			// to the next ';' in the body and parse any buffalo.models.field block.
			rest := msg.body[flIdx[1]:]
			semiIdx := strings.Index(rest, ";")
			if semiIdx >= 0 {
				fieldTail := rest[:semiIdx+1]
				if strings.Contains(fieldTail, "buffalo.models.field") {
					fieldMatch := fieldAnnotationRe.FindStringSubmatch(fieldTail)
					if fieldMatch != nil {
						if err := parseFieldOptions(fieldMatch[1], &fd); err != nil {
							return nil, fmt.Errorf("%s: %s.%s: %w", filePath, msg.name, fd.Name, err)
						}
					}
				}
			}

			md.Fields = append(md.Fields, fd)
		}

		results = append(results, md)
	}

	return results, nil
}

// ExtractAllMessages scans a full proto file and returns ModelDefs for EVERY
// message, regardless of whether it has buffalo.models annotations.
// This is the "generate_models_from_proto" mode:  plain proto messages
// become models with auto-derived settings.  Annotated messages get their
// annotations applied on top.
//
// Service definitions are skipped.  Nested enums are extracted into ModelDef.Enums.
// map<K,V> fields are represented with IsMap=true, MapKeyType, MapValueType.
func ExtractAllMessages(content, filePath string) ([]ModelDef, error) {
	pkg := extractPackage(content)
	messages := extractMessageBlocks(content)

	var results []ModelDef
	for _, msg := range messages {
		// Skip if this is inside a service block (heuristic: check if name
		// matches a Request/Response that is NOT a standalone message)
		// — actually, extractMessageBlocks already only matches "message" keyword,
		// so services are not captured.

		md := ModelDef{
			MessageName: msg.name,
			Package:     pkg,
			FilePath:    filePath,
			Description: msg.comment,
			Fields:      []FieldDef{},
			Enums:       []EnumDef{},
		}

		// If this message also has buffalo.models annotations, apply them
		modelMatch := modelAnnotationRe.FindStringSubmatch(msg.body)
		if modelMatch != nil {
			if err := parseModelOptions(modelMatch[1], &md); err != nil {
				return nil, fmt.Errorf("%s: message %s: %w", filePath, msg.name, err)
			}
		}

		// Extract nested enums
		md.Enums = extractEnums(msg.body)

		// Build a set of enum type names for resolving field types
		enumNames := map[string]bool{}
		for _, e := range md.Enums {
			enumNames[e.Name] = true
		}

		// Extract map fields first (they have special syntax)
		mapFieldPositions := map[string]bool{} // field name → consumed
		mapMatches := mapFieldRe.FindAllStringSubmatch(msg.body, -1)
		for _, mf := range mapMatches {
			keyType := mf[1]
			valType := mf[2]
			fName := mf[3]
			fNum, _ := strconv.Atoi(mf[4])

			fd := FieldDef{
				Name:         fName,
				ProtoType:    "map",
				Number:       fNum,
				IsMap:        true,
				MapKeyType:   keyType,
				MapValueType: valType,
			}

			// Extract field comment
			fd.Description = extractFieldComment(msg.body, fName)

			// If annotated, apply field options
			applyFieldAnnotation(msg.body, fName, &fd)

			md.Fields = append(md.Fields, fd)
			mapFieldPositions[fName] = true
		}

		// Extract oneof blocks and remember which fields belong to which group
		oneofFields := extractOneofFields(msg.body)

		// Extract regular fields
		fieldLines := fieldLineRe.FindAllStringSubmatchIndex(msg.body, -1)
		for _, flIdx := range fieldLines {
			fl := []string{
				msg.body[flIdx[0]:flIdx[1]],
				"",
				"",
				"",
				"",
			}
			if flIdx[2] >= 0 {
				fl[1] = msg.body[flIdx[2]:flIdx[3]]
			}
			fl[2] = msg.body[flIdx[4]:flIdx[5]]
			fl[3] = msg.body[flIdx[6]:flIdx[7]]
			fl[4] = msg.body[flIdx[8]:flIdx[9]]

			fieldName := fl[3]

			// Skip if already captured as a map field
			if mapFieldPositions[fieldName] {
				continue
			}

			qualifier := strings.TrimSpace(fl[1])
			fd := FieldDef{
				Repeated:  qualifier == "repeated",
				Nullable:  qualifier == "optional",
				ProtoType: fl[2],
				Name:      fieldName,
			}
			if n, err := strconv.Atoi(fl[4]); err == nil {
				fd.Number = n
			}

			// If field type is a nested enum, mark ProtoType accordingly
			if enumNames[fd.ProtoType] {
				fd.ProtoType = "int32" // enums map to int in generated models
				fd.Comment = fmt.Sprintf("enum %s", fl[2])
			}

			// Assign oneof group
			if group, ok := oneofFields[fieldName]; ok {
				fd.OneofGroup = group
				fd.Nullable = true // oneof fields are implicitly optional
			}

			// Extract field comment
			fd.Description = extractFieldComment(msg.body, fieldName)

			// Check for buffalo.models.field annotation
			applyFieldAnnotation(msg.body, fieldName, &fd)

			// Also check inline annotation (original logic)
			rest := msg.body[flIdx[1]:]
			semiIdx := strings.Index(rest, ";")
			if semiIdx >= 0 {
				fieldTail := rest[:semiIdx+1]
				if strings.Contains(fieldTail, "buffalo.models.field") {
					fieldMatch := fieldAnnotationRe.FindStringSubmatch(fieldTail)
					if fieldMatch != nil {
						if err := parseFieldOptions(fieldMatch[1], &fd); err != nil {
							return nil, fmt.Errorf("%s: %s.%s: %w", filePath, msg.name, fd.Name, err)
						}
					}
				}
			}

			md.Fields = append(md.Fields, fd)
		}

		results = append(results, md)
	}

	return results, nil
}

// extractEnums parses enum blocks within a message body.
func extractEnums(body string) []EnumDef {
	var enums []EnumDef

	locs := enumBlockRe.FindAllStringSubmatchIndex(body, -1)
	for _, loc := range locs {
		name := body[loc[2]:loc[3]]
		braceStart := loc[1]

		// Find matching }
		depth := 1
		pos := braceStart
		for pos < len(body) && depth > 0 {
			switch body[pos] {
			case '{':
				depth++
			case '}':
				depth--
			}
			pos++
		}

		if depth == 0 {
			enumBody := body[braceStart : pos-1]
			ed := EnumDef{Name: name}

			valMatches := enumValueRe.FindAllStringSubmatch(enumBody, -1)
			for _, vm := range valMatches {
				num, _ := strconv.ParseInt(vm[2], 10, 32)
				ed.Values = append(ed.Values, EnumValue{
					Name:   vm[1],
					Number: int32(num),
				})
			}

			enums = append(enums, ed)
		}
	}

	return enums
}

// extractOneofFields parses oneof blocks and returns field_name → oneof_group_name.
func extractOneofFields(body string) map[string]string {
	result := map[string]string{}

	locs := oneofBlockRe.FindAllStringSubmatchIndex(body, -1)
	for _, loc := range locs {
		groupName := body[loc[2]:loc[3]]
		braceStart := loc[1]

		depth := 1
		pos := braceStart
		for pos < len(body) && depth > 0 {
			switch body[pos] {
			case '{':
				depth++
			case '}':
				depth--
			}
			pos++
		}

		if depth == 0 {
			oneofBody := body[braceStart : pos-1]
			fieldMatches := fieldLineRe.FindAllStringSubmatch(oneofBody, -1)
			for _, fm := range fieldMatches {
				fieldName := fm[3]
				result[fieldName] = groupName
			}
		}
	}

	return result
}

// extractFieldComment extracts the comment on the line(s) above a field.
func extractFieldComment(body, fieldName string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		// Find the line containing the field definition
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, fieldName) &&
			(strings.Contains(trimmed, "=") || strings.HasPrefix(trimmed, "map<")) {
			// Collect comment lines above it
			var comments []string
			for j := i - 1; j >= 0; j-- {
				cl := strings.TrimSpace(lines[j])
				if strings.HasPrefix(cl, "//") {
					comments = append([]string{strings.TrimSpace(strings.TrimPrefix(cl, "//"))}, comments...)
				} else if cl == "" {
					break
				} else {
					break
				}
			}
			if len(comments) > 0 {
				return strings.Join(comments, " ")
			}
			// Also check inline comment
			if idx := strings.Index(line, "//"); idx >= 0 {
				return strings.TrimSpace(line[idx+2:])
			}
			return ""
		}
	}
	return ""
}

// applyFieldAnnotation finds and applies a buffalo.models.field annotation for a named field.
func applyFieldAnnotation(body, fieldName string, fd *FieldDef) {
	// Find the line with this field and check for annotation
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, fieldName) && strings.Contains(trimmed, "buffalo.models.field") {
			fieldMatch := fieldAnnotationRe.FindStringSubmatch(trimmed)
			if fieldMatch != nil {
				_ = parseFieldOptions(fieldMatch[1], fd)
			}
		}
	}
}

// ──────────────────────────────────────────────────────────────────
// Internal: proto structure extraction
// ──────────────────────────────────────────────────────────────────

type rawMessage struct {
	name    string
	body    string
	comment string // leading comment block
}

func extractPackage(content string) string {
	m := packageRe.FindStringSubmatch(content)
	if m != nil {
		return m[1]
	}
	return ""
}

// extractMessageBlocks finds top-level message blocks with their bodies.
func extractMessageBlocks(content string) []rawMessage {
	locs := messageBlockRe.FindAllStringSubmatchIndex(content, -1)
	var messages []rawMessage

	for _, loc := range locs {
		name := content[loc[2]:loc[3]]
		bodyStart := loc[1] // position after "message Name {"

		// Find the matching closing brace
		depth := 1
		pos := bodyStart
		for pos < len(content) && depth > 0 {
			switch content[pos] {
			case '{':
				depth++
			case '}':
				depth--
			}
			pos++
		}

		if depth == 0 {
			// body is between the opening { (at bodyStart-1) and closing }
			body := content[bodyStart:pos]

			// Extract leading comment (lines immediately before "message Name {")
			comment := extractLeadingComment(content, loc[0])

			messages = append(messages, rawMessage{name: name, body: body, comment: comment})
		}
	}

	return messages
}

// extractLeadingComment extracts // comment lines immediately before a position.
func extractLeadingComment(content string, pos int) string {
	// Walk backwards from pos to find consecutive comment lines
	lines := strings.Split(content[:pos], "\n")
	var commentLines []string
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			// Skip one empty line
			if len(commentLines) == 0 {
				continue
			}
			break
		}
		if strings.HasPrefix(trimmed, "//") {
			commentLines = append([]string{strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))}, commentLines...)
		} else {
			break
		}
	}
	return strings.Join(commentLines, " ")
}

// ──────────────────────────────────────────────────────────────────
// Internal: parse model-level options
// ──────────────────────────────────────────────────────────────────

func parseModelOptions(body string, md *ModelDef) error {
	kvs := kvRe.FindAllStringSubmatch(body, -1)
	for _, kv := range kvs {
		key := kv[1]
		val := strings.Trim(kv[2], `"' `)

		switch key {
		case "name":
			md.Name = val
		case "table_name":
			md.TableName = val
		case "schema":
			md.Schema = val
		case "description":
			md.Description = val
		case "abstract":
			md.Abstract = parseBool(val)
		case "extends":
			md.Extends = val
		case "soft_delete":
			md.SoftDelete = parseBool(val)
		case "timestamps":
			md.Timestamps = parseBool(val)
		case "deprecated":
			md.Deprecated = parseBool(val)
		case "deprecated_message":
			md.DeprecatedMessage = val
		case "tags":
			md.Tags = parseStringList(kv[2])
		case "mixins":
			md.Mixins = parseStringList(kv[2])
		case "generate":
			md.Generate = parseStringList(kv[2])
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────
// Internal: parse field-level options
// ──────────────────────────────────────────────────────────────────

func parseFieldOptions(body string, fd *FieldDef) error {
	kvs := kvRe.FindAllStringSubmatch(body, -1)
	for _, kv := range kvs {
		key := kv[1]
		rawVal := kv[2]
		val := strings.Trim(rawVal, `"' `)

		switch key {
		case "alias":
			fd.Alias = val
		case "description":
			fd.Description = val
		case "primary_key":
			fd.PrimaryKey = parseBool(val)
		case "auto_increment":
			fd.AutoIncrement = parseBool(val)
		case "nullable":
			fd.Nullable = parseBool(val)
		case "unique":
			fd.Unique = parseBool(val)
		case "default_value":
			fd.DefaultValue = val
		case "max_length":
			fd.MaxLength = parseInt32(val)
		case "min_length":
			fd.MinLength = parseInt32(val)
		case "precision":
			fd.Precision = parseInt32(val)
		case "scale":
			fd.Scale = parseInt32(val)
		case "custom_type":
			fd.CustomType = val
		case "db_type":
			fd.DBType = val
		case "visibility":
			fd.Visibility = parseVisibility(val)
		case "behavior":
			fd.Behavior = parseBehavior(val)
		case "sensitive":
			fd.Sensitive = parseBool(val)
		case "deprecated":
			fd.Deprecated = parseBool(val)
		case "deprecated_message":
			fd.DeprecatedMessage = val
		case "index":
			fd.Index = parseBool(val)
		case "index_type":
			fd.IndexType = parseIndexType(val)
		case "json_name":
			fd.JSONName = val
		case "xml_name":
			fd.XMLName = val
		case "omit_empty":
			fd.OmitEmpty = parseBool(val)
		case "example":
			fd.Example = val
		case "comment":
			fd.Comment = val
		case "auto_generate":
			fd.AutoGenerate = parseBool(val)
		case "auto_now":
			fd.AutoNow = parseBool(val)
		case "auto_now_add":
			fd.AutoNowAdd = parseBool(val)
		case "sequence":
			fd.Sequence = val
		case "collation":
			fd.Collation = val
		case "ignore":
			fd.Ignore = parseBool(val)
		case "db_ignore":
			fd.DBIgnore = parseBool(val)
		case "api_ignore":
			fd.APIIgnore = parseBool(val)
		case "tags":
			fd.Tags = parseStringList(rawVal)
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────────────
// Internal: value parsers
// ──────────────────────────────────────────────────────────────────

func parseBool(s string) bool {
	return s == "true" || s == "1"
}

func parseInt32(s string) int32 {
	v, _ := strconv.ParseInt(s, 10, 32)
	return int32(v)
}

func parseStringList(s string) []string {
	matches := listValueRe.FindAllStringSubmatch(s, -1)
	var result []string
	for _, m := range matches {
		result = append(result, m[1])
	}
	return result
}

func parseVisibility(s string) FieldVisibility {
	switch strings.ToUpper(s) {
	case "PUBLIC", "1":
		return VisibilityPublic
	case "INTERNAL", "2":
		return VisibilityInternal
	case "EXTERNAL", "3":
		return VisibilityExternal
	case "PRIVATE", "4":
		return VisibilityPrivate
	case "PROTECTED", "5":
		return VisibilityProtected
	default:
		return VisibilityDefault
	}
}

func parseBehavior(s string) FieldBehavior {
	switch strings.ToUpper(s) {
	case "READONLY", "1":
		return BehaviorReadOnly
	case "WRITEONLY", "2":
		return BehaviorWriteOnly
	case "IMMUTABLE", "3":
		return BehaviorImmutable
	case "COMPUTED", "4":
		return BehaviorComputed
	case "VIRTUAL", "5":
		return BehaviorVirtual
	case "OUTPUT_ONLY", "6":
		return BehaviorOutputOnly
	case "INPUT_ONLY", "7":
		return BehaviorInputOnly
	default:
		return BehaviorDefault
	}
}

func parseIndexType(s string) IndexType {
	switch strings.ToUpper(s) {
	case "BTREE", "1":
		return IndexBTree
	case "HASH", "2":
		return IndexHash
	case "GIN", "3":
		return IndexGIN
	case "GIST", "4":
		return IndexGIST
	case "BRIN", "5":
		return IndexBRIN
	default:
		return IndexDefault
	}
}
