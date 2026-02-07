package lsp

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/massonsky/buffalo/internal/embedded"
	"github.com/massonsky/buffalo/internal/permissions"
	"github.com/massonsky/buffalo/pkg/logger"
)

// ProtoAnalyzer provides proto file analysis for LSP.
type ProtoAnalyzer struct {
	log              *logger.Logger
	embeddedProtos   map[string]string // path -> content
	permissionParser *permissions.Parser
	validationRules  map[string]ValidationRuleInfo
}

// ValidationRuleInfo describes a Buffalo validation rule.
type ValidationRuleInfo struct {
	Name        string
	Type        string // string, int32, double, etc.
	Description string
	Example     string
}

// PermissionAnnotationInfo describes a Buffalo permission annotation.
type PermissionAnnotationInfo struct {
	Name        string
	Description string
	Example     string
}

// NewProtoAnalyzer creates a new proto analyzer.
func NewProtoAnalyzer(log *logger.Logger) *ProtoAnalyzer {
	a := &ProtoAnalyzer{
		log:              log,
		embeddedProtos:   make(map[string]string),
		permissionParser: permissions.NewParser(),
		validationRules:  initValidationRules(),
	}
	a.loadEmbeddedProtos()
	return a
}

// loadEmbeddedProtos loads all embedded proto files into memory.
func (a *ProtoAnalyzer) loadEmbeddedProtos() {
	err := fs.WalkDir(embedded.ProtoFS, "proto", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".proto") {
			return nil
		}
		data, err := embedded.ProtoFS.ReadFile(path)
		if err != nil {
			return err
		}
		// Store with normalized import path (e.g., "buffalo/validate/validate.proto")
		importPath := strings.TrimPrefix(path, "proto/")
		a.embeddedProtos[importPath] = string(data)
		return nil
	})
	if err != nil {
		a.log.Warn("Failed to load embedded protos", logger.Error(err))
	}
}

// GetEmbeddedProto returns content of an embedded proto file.
func (a *ProtoAnalyzer) GetEmbeddedProto(importPath string) (string, bool) {
	content, ok := a.embeddedProtos[importPath]
	return content, ok
}

// ListEmbeddedProtos returns all embedded proto file paths.
func (a *ProtoAnalyzer) ListEmbeddedProtos() []string {
	paths := make([]string, 0, len(a.embeddedProtos))
	for path := range a.embeddedProtos {
		paths = append(paths, path)
	}
	return paths
}

// initValidationRules initializes the validation rules documentation.
func initValidationRules() map[string]ValidationRuleInfo {
	return map[string]ValidationRuleInfo{
		// String rules
		"string.min_len":   {Name: "min_len", Type: "string", Description: "Minimum string length", Example: `min_len: 1`},
		"string.max_len":   {Name: "max_len", Type: "string", Description: "Maximum string length", Example: `max_len: 255`},
		"string.pattern":   {Name: "pattern", Type: "string", Description: "Regex pattern to match", Example: `pattern: "^[a-z]+$"`},
		"string.email":     {Name: "email", Type: "string", Description: "Must be a valid email address", Example: `email: true`},
		"string.uri":       {Name: "uri", Type: "string", Description: "Must be a valid URI", Example: `uri: true`},
		"string.uuid":      {Name: "uuid", Type: "string", Description: "Must be a valid UUID", Example: `uuid: true`},
		"string.ip":        {Name: "ip", Type: "string", Description: "Must be a valid IP address", Example: `ip: true`},
		"string.hostname":  {Name: "hostname", Type: "string", Description: "Must be a valid hostname", Example: `hostname: true`},
		"string.not_empty": {Name: "not_empty", Type: "string", Description: "String must not be empty", Example: `not_empty: true`},
		"string.prefix":    {Name: "prefix", Type: "string", Description: "String must start with prefix", Example: `prefix: "https://"`},
		"string.suffix":    {Name: "suffix", Type: "string", Description: "String must end with suffix", Example: `suffix: ".com"`},
		"string.contains":  {Name: "contains", Type: "string", Description: "String must contain substring", Example: `contains: "@"`},
		"string.in":        {Name: "in", Type: "string", Description: "Value must be in the list", Example: `in: ["a", "b", "c"]`},
		"string.not_in":    {Name: "not_in", Type: "string", Description: "Value must not be in the list", Example: `not_in: ["x", "y"]`},
		// Numeric rules
		"int32.gt":     {Name: "gt", Type: "int32", Description: "Value must be greater than", Example: `gt: 0`},
		"int32.gte":    {Name: "gte", Type: "int32", Description: "Value must be greater than or equal", Example: `gte: 1`},
		"int32.lt":     {Name: "lt", Type: "int32", Description: "Value must be less than", Example: `lt: 100`},
		"int32.lte":    {Name: "lte", Type: "int32", Description: "Value must be less than or equal", Example: `lte: 99`},
		"int32.in":     {Name: "in", Type: "int32", Description: "Value must be in the list", Example: `in: [1, 2, 3]`},
		"int32.not_in": {Name: "not_in", Type: "int32", Description: "Value must not be in the list", Example: `not_in: [0, -1]`},
		// Double/float rules
		"double.gt":  {Name: "gt", Type: "double", Description: "Value must be greater than", Example: `gt: 0.0`},
		"double.gte": {Name: "gte", Type: "double", Description: "Value must be greater than or equal", Example: `gte: -90.0`},
		"double.lt":  {Name: "lt", Type: "double", Description: "Value must be less than", Example: `lt: 100.0`},
		"double.lte": {Name: "lte", Type: "double", Description: "Value must be less than or equal", Example: `lte: 90.0`},
		// Repeated rules
		"repeated.min_items": {Name: "min_items", Type: "repeated", Description: "Minimum number of items", Example: `min_items: 1`},
		"repeated.max_items": {Name: "max_items", Type: "repeated", Description: "Maximum number of items", Example: `max_items: 100`},
		"repeated.unique":    {Name: "unique", Type: "repeated", Description: "All items must be unique", Example: `unique: true`},
		// Map rules
		"map.min_pairs": {Name: "min_pairs", Type: "map", Description: "Minimum number of key-value pairs", Example: `min_pairs: 1`},
		"map.max_pairs": {Name: "max_pairs", Type: "map", Description: "Maximum number of key-value pairs", Example: `max_pairs: 50`},
		// Message rules
		"message.required": {Name: "required", Type: "message", Description: "Field must be set", Example: `required: true`},
		// Timestamp rules
		"timestamp.gt_now":         {Name: "gt_now", Type: "timestamp", Description: "Timestamp must be in the future", Example: `gt_now: true`},
		"timestamp.lt_now":         {Name: "lt_now", Type: "timestamp", Description: "Timestamp must be in the past", Example: `lt_now: true`},
		"timestamp.within_seconds": {Name: "within_seconds", Type: "timestamp", Description: "Timestamp must be within N seconds of now", Example: `within_seconds: 3600`},
	}
}

// ProtoSymbol represents a symbol in a proto file.
type ProtoSymbol struct {
	Name          string
	Kind          SymbolKind
	Range         Range
	Children      []ProtoSymbol
	Type          string // For fields: the field type
	Documentation string
}

// Regex patterns for proto parsing
var (
	syntaxPattern     = regexp.MustCompile(`^\s*syntax\s*=\s*["']([^"']+)["']\s*;`)
	packagePattern    = regexp.MustCompile(`^\s*package\s+([a-zA-Z_][a-zA-Z0-9_.]*)\s*;`)
	importPattern     = regexp.MustCompile(`^\s*import\s+(?:(weak|public)\s+)?["']([^"']+)["']\s*;`)
	optionPattern     = regexp.MustCompile(`^\s*option\s+([a-zA-Z_][a-zA-Z0-9_.]*)\s*=`)
	messagePattern    = regexp.MustCompile(`^\s*message\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\{?`)
	enumPattern       = regexp.MustCompile(`^\s*enum\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\{?`)
	servicePattern    = regexp.MustCompile(`^\s*service\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\{?`)
	rpcPattern        = regexp.MustCompile(`^\s*rpc\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	fieldPattern      = regexp.MustCompile(`^\s*(repeated|optional|required)?\s*([a-zA-Z_][a-zA-Z0-9_.]*)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(\d+)`)
	enumValuePattern  = regexp.MustCompile(`^\s*([A-Z_][A-Z0-9_]*)\s*=\s*(-?\d+)`)
	oneofPattern      = regexp.MustCompile(`^\s*oneof\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\{?`)
	mapPattern        = regexp.MustCompile(`^\s*map\s*<\s*([^,]+)\s*,\s*([^>]+)\s*>\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(\d+)`)
	commentPattern    = regexp.MustCompile(`^\s*//(.*)$`)
	blockCommentStart = regexp.MustCompile(`^\s*/\*`)
	blockCommentEnd   = regexp.MustCompile(`\*/`)

	// Buffalo annotation patterns
	buffaloOptionPattern = regexp.MustCompile(`\[\s*\(\s*buffalo\.`)
	validationPattern    = regexp.MustCompile(`buffalo\.validate\.rules`)
	permissionPattern    = regexp.MustCompile(`buffalo\.permissions`)
)

// Analyze analyzes a document and returns diagnostics.
func (a *ProtoAnalyzer) Analyze(doc *Document) []Diagnostic {
	diagnostics := []Diagnostic{}

	if !strings.HasSuffix(string(doc.URI), ".proto") {
		return diagnostics
	}

	inBlockComment := false
	braceCount := 0
	hasSyntax := false
	hasPackage := false
	currentContext := ""

	for lineNum, line := range doc.Lines {
		// Handle block comments
		if inBlockComment {
			if blockCommentEnd.MatchString(line) {
				inBlockComment = false
			}
			continue
		}

		if blockCommentStart.MatchString(line) && !blockCommentEnd.MatchString(line) {
			inBlockComment = true
			continue
		}

		// Skip empty lines and single-line comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Track braces
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")

		// Check syntax
		if syntaxPattern.MatchString(line) {
			hasSyntax = true
			matches := syntaxPattern.FindStringSubmatch(line)
			if len(matches) > 1 && matches[1] != "proto3" && matches[1] != "proto2" {
				diagnostics = append(diagnostics, Diagnostic{
					Range:    lineRange(lineNum, line),
					Severity: SeverityError,
					Source:   "buffalo-lsp",
					Message:  fmt.Sprintf("Invalid syntax: %s. Expected 'proto2' or 'proto3'", matches[1]),
				})
			}
		}

		// Check package
		if packagePattern.MatchString(line) {
			hasPackage = true
		}

		// Track context
		if messagePattern.MatchString(line) {
			currentContext = "message"
		} else if servicePattern.MatchString(line) {
			currentContext = "service"
		} else if enumPattern.MatchString(line) {
			currentContext = "enum"
		}

		// Validate field numbers
		if fieldPattern.MatchString(line) {
			matches := fieldPattern.FindStringSubmatch(line)
			if len(matches) > 4 {
				fieldNum := matches[4]
				var num int
				fmt.Sscanf(fieldNum, "%d", &num)
				if num <= 0 {
					diagnostics = append(diagnostics, Diagnostic{
						Range:    lineRange(lineNum, line),
						Severity: SeverityError,
						Source:   "buffalo-lsp",
						Message:  "Field number must be positive",
					})
				}
				if num >= 19000 && num <= 19999 {
					diagnostics = append(diagnostics, Diagnostic{
						Range:    lineRange(lineNum, line),
						Severity: SeverityError,
						Source:   "buffalo-lsp",
						Message:  "Field numbers 19000-19999 are reserved for protobuf implementation",
					})
				}
			}
		}

		// Check for common mistakes
		if strings.Contains(line, "= 0") && currentContext != "enum" {
			if fieldPattern.MatchString(line) {
				diagnostics = append(diagnostics, Diagnostic{
					Range:    lineRange(lineNum, line),
					Severity: SeverityError,
					Source:   "buffalo-lsp",
					Message:  "Field number 0 is not allowed (except for enum default values)",
				})
			}
		}

		// Lint: check naming conventions
		if fieldPattern.MatchString(line) {
			matches := fieldPattern.FindStringSubmatch(line)
			if len(matches) > 3 {
				fieldName := matches[3]
				if !isSnakeCase(fieldName) {
					diagnostics = append(diagnostics, Diagnostic{
						Range:    lineRange(lineNum, line),
						Severity: SeverityHint,
						Source:   "buffalo-lsp",
						Message:  fmt.Sprintf("Field '%s' should use snake_case naming", fieldName),
						Tags:     []DiagnosticTag{DiagnosticTagUnnecessary},
					})
				}
			}
		}

		// Check message naming (should be PascalCase)
		if messagePattern.MatchString(line) {
			matches := messagePattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				msgName := matches[1]
				if !isPascalCase(msgName) {
					diagnostics = append(diagnostics, Diagnostic{
						Range:    lineRange(lineNum, line),
						Severity: SeverityHint,
						Source:   "buffalo-lsp",
						Message:  fmt.Sprintf("Message '%s' should use PascalCase naming", msgName),
					})
				}
			}
		}

		// Check enum value naming (should be UPPER_SNAKE_CASE)
		if enumValuePattern.MatchString(line) && currentContext == "enum" {
			matches := enumValuePattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				valueName := matches[1]
				if !isUpperSnakeCase(valueName) {
					diagnostics = append(diagnostics, Diagnostic{
						Range:    lineRange(lineNum, line),
						Severity: SeverityHint,
						Source:   "buffalo-lsp",
						Message:  fmt.Sprintf("Enum value '%s' should use UPPER_SNAKE_CASE naming", valueName),
					})
				}
			}
		}
	}

	// Global checks
	if !hasSyntax {
		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: SeverityWarning,
			Source:   "buffalo-lsp",
			Message:  "Missing syntax declaration. Consider adding: syntax = \"proto3\";",
		})
	}

	if !hasPackage {
		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: SeverityWarning,
			Source:   "buffalo-lsp",
			Message:  "Missing package declaration",
		})
	}

	if braceCount != 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: len(doc.Lines) - 1, Character: 0},
				End:   Position{Line: len(doc.Lines) - 1, Character: 0},
			},
			Severity: SeverityError,
			Source:   "buffalo-lsp",
			Message:  "Mismatched braces",
		})
	}

	// Syntax diagnostics (deep analysis)
	diagnostics = append(diagnostics, a.SyntaxDiagnostics(doc)...)

	// Buffalo-specific diagnostics
	diagnostics = append(diagnostics, a.analyzeBuffaloAnnotations(doc)...)
	diagnostics = append(diagnostics, a.analyzePermissions(doc)...)
	diagnostics = append(diagnostics, a.analyzeImports(doc)...)

	return deduplicateDiagnostics(diagnostics)
}

// analyzeBuffaloAnnotations checks Buffalo validation annotations.
func (a *ProtoAnalyzer) analyzeBuffaloAnnotations(doc *Document) []Diagnostic {
	diagnostics := []Diagnostic{}

	inValidation := false
	validationStart := 0
	braceCount := 0

	for lineNum, line := range doc.Lines {
		// Check for buffalo.validate.rules usage
		if strings.Contains(line, "buffalo.validate.rules") {
			inValidation = true
			validationStart = lineNum
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")

			// Check if validation import is present
			hasImport := false
			for _, l := range doc.Lines {
				if strings.Contains(l, `"buffalo/validate/validate.proto"`) ||
					strings.Contains(l, `"buffalo/validate.proto"`) {
					hasImport = true
					break
				}
			}
			if !hasImport {
				diagnostics = append(diagnostics, Diagnostic{
					Range:    lineRange(lineNum, line),
					Severity: SeverityWarning,
					Source:   "buffalo-lsp",
					Message:  "Missing import for Buffalo validation. Add: import \"buffalo/validate/validate.proto\";",
				})
			}
		}

		if inValidation {
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inValidation = false
			}

			// Check for unknown validation rules
			a.checkValidationRules(line, lineNum, &diagnostics)
		}
	}

	// Check for unterminated validation block
	if inValidation {
		diagnostics = append(diagnostics, Diagnostic{
			Range:    lineRange(validationStart, doc.Lines[validationStart]),
			Severity: SeverityError,
			Source:   "buffalo-lsp",
			Message:  "Unterminated validation annotation",
		})
	}

	return diagnostics
}

// checkValidationRules validates individual validation rules.
func (a *ProtoAnalyzer) checkValidationRules(line string, lineNum int, diagnostics *[]Diagnostic) {
	// Check for conflicting rules
	if strings.Contains(line, "email: true") && strings.Contains(line, "uri: true") {
		*diagnostics = append(*diagnostics, Diagnostic{
			Range:    lineRange(lineNum, line),
			Severity: SeverityWarning,
			Source:   "buffalo-lsp",
			Message:  "Conflicting validation: email and uri cannot both be true",
		})
	}

	// Check for invalid range
	if strings.Contains(line, "min_len") && strings.Contains(line, "max_len") {
		minMatch := regexp.MustCompile(`min_len:\s*(\d+)`).FindStringSubmatch(line)
		maxMatch := regexp.MustCompile(`max_len:\s*(\d+)`).FindStringSubmatch(line)
		if len(minMatch) > 1 && len(maxMatch) > 1 {
			var min, max int
			fmt.Sscanf(minMatch[1], "%d", &min)
			fmt.Sscanf(maxMatch[1], "%d", &max)
			if min > max {
				*diagnostics = append(*diagnostics, Diagnostic{
					Range:    lineRange(lineNum, line),
					Severity: SeverityError,
					Source:   "buffalo-lsp",
					Message:  fmt.Sprintf("Invalid range: min_len (%d) > max_len (%d)", min, max),
				})
			}
		}
	}

	// Check for gt/gte and lt/lte conflicts
	if (strings.Contains(line, "gt:") || strings.Contains(line, "gte:")) &&
		(strings.Contains(line, "lt:") || strings.Contains(line, "lte:")) {
		gtMatch := regexp.MustCompile(`gt[e]?:\s*(-?\d+(?:\.\d+)?)`).FindStringSubmatch(line)
		ltMatch := regexp.MustCompile(`lt[e]?:\s*(-?\d+(?:\.\d+)?)`).FindStringSubmatch(line)
		if len(gtMatch) > 1 && len(ltMatch) > 1 {
			var gt, lt float64
			fmt.Sscanf(gtMatch[1], "%f", &gt)
			fmt.Sscanf(ltMatch[1], "%f", &lt)
			if gt >= lt {
				*diagnostics = append(*diagnostics, Diagnostic{
					Range:    lineRange(lineNum, line),
					Severity: SeverityError,
					Source:   "buffalo-lsp",
					Message:  fmt.Sprintf("Invalid range: gt/gte value (%.2f) >= lt/lte value (%.2f)", gt, lt),
				})
			}
		}
	}
}

// analyzePermissions checks Buffalo permission annotations.
func (a *ProtoAnalyzer) analyzePermissions(doc *Document) []Diagnostic {
	diagnostics := []Diagnostic{}
	inService := false
	hasPermissionImport := false
	serviceHasPermissions := make(map[string]bool)
	currentService := ""

	for _, line := range doc.Lines {
		if strings.Contains(line, `"buffalo/permissions"`) ||
			strings.Contains(line, `"buffalo/permissions.proto"`) {
			hasPermissionImport = true
			break
		}
	}

	for lineNum, line := range doc.Lines {
		// Track service context
		if servicePattern.MatchString(line) {
			inService = true
			matches := servicePattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentService = matches[1]
			}
		}

		if inService && strings.Contains(line, "}") && !strings.Contains(line, "{") {
			// Check if service has any permissions defined
			if !serviceHasPermissions[currentService] {
				// This is just informational, not an error
			}
			inService = false
			currentService = ""
		}

		// Check for permission annotations
		if strings.Contains(line, "buffalo.permissions") {
			serviceHasPermissions[currentService] = true

			if !hasPermissionImport {
				diagnostics = append(diagnostics, Diagnostic{
					Range:    lineRange(lineNum, line),
					Severity: SeverityWarning,
					Source:   "buffalo-lsp",
					Message:  "Using buffalo.permissions without import. Consider importing permissions proto.",
				})
			}

			// Validate permission annotation structure
			if strings.Contains(line, "buffalo.permissions.required") ||
				strings.Contains(line, "buffalo.permissions.roles") {
				// Check for empty roles
				if strings.Contains(line, "roles: []") {
					diagnostics = append(diagnostics, Diagnostic{
						Range:    lineRange(lineNum, line),
						Severity: SeverityWarning,
						Source:   "buffalo-lsp",
						Message:  "Empty roles list - this effectively denies all access",
					})
				}
			}

			// Check for public + roles conflict
			if strings.Contains(line, "public: true") && strings.Contains(line, "roles:") {
				diagnostics = append(diagnostics, Diagnostic{
					Range:    lineRange(lineNum, line),
					Severity: SeverityWarning,
					Source:   "buffalo-lsp",
					Message:  "Conflicting permission: 'public: true' with 'roles' defined - roles will be ignored",
				})
			}

			// Check for require_mfa on public endpoint
			if strings.Contains(line, "public: true") && strings.Contains(line, "require_mfa: true") {
				diagnostics = append(diagnostics, Diagnostic{
					Range:    lineRange(lineNum, line),
					Severity: SeverityError,
					Source:   "buffalo-lsp",
					Message:  "Invalid permission: cannot require MFA on public endpoint",
				})
			}
		}

		// Check RPC methods for permission hints
		if rpcPattern.MatchString(line) && inService {
			rpcName := ""
			if matches := rpcPattern.FindStringSubmatch(line); len(matches) > 1 {
				rpcName = matches[1]
			}

			// Look ahead for permission annotation
			hasPermission := false
			for i := lineNum; i < len(doc.Lines) && i < lineNum+5; i++ {
				if strings.Contains(doc.Lines[i], "buffalo.permissions") {
					hasPermission = true
					break
				}
				if strings.Contains(doc.Lines[i], "rpc ") && i != lineNum {
					break
				}
			}

			// Suggest permissions for sensitive operations
			if !hasPermission {
				lowerName := strings.ToLower(rpcName)
				if strings.HasPrefix(lowerName, "delete") ||
					strings.HasPrefix(lowerName, "remove") ||
					strings.HasPrefix(lowerName, "update") ||
					strings.HasPrefix(lowerName, "create") {
					diagnostics = append(diagnostics, Diagnostic{
						Range:    lineRange(lineNum, line),
						Severity: SeverityHint,
						Source:   "buffalo-lsp",
						Message:  fmt.Sprintf("Consider adding permission annotation for '%s' method", rpcName),
					})
				}
			}
		}
	}

	return diagnostics
}

// analyzeImports checks import statements and suggests Buffalo imports.
func (a *ProtoAnalyzer) analyzeImports(doc *Document) []Diagnostic {
	diagnostics := []Diagnostic{}

	hasValidation := false
	hasPermissions := false
	hasValidationImport := false
	hasPermissionsImport := false

	for _, line := range doc.Lines {
		if strings.Contains(line, "buffalo.validate") {
			hasValidation = true
		}
		if strings.Contains(line, "buffalo.permissions") {
			hasPermissions = true
		}
		if strings.Contains(line, `"buffalo/validate`) {
			hasValidationImport = true
		}
		if strings.Contains(line, `"buffalo/permissions`) {
			hasPermissionsImport = true
		}
	}

	// Check for missing imports
	for lineNum, line := range doc.Lines {
		if importPattern.MatchString(line) {
			matches := importPattern.FindStringSubmatch(line)
			if len(matches) > 2 {
				importPath := matches[2]
				// Check if it's a Buffalo embedded proto
				if _, exists := a.embeddedProtos[importPath]; exists {
					// Valid embedded import
					continue
				}
				// Check for common typos
				if strings.Contains(importPath, "bufalo") || strings.Contains(importPath, "baffalo") {
					diagnostics = append(diagnostics, Diagnostic{
						Range:    lineRange(lineNum, line),
						Severity: SeverityError,
						Source:   "buffalo-lsp",
						Message:  fmt.Sprintf("Possible typo in import path: %s", importPath),
					})
				}
			}
		}
	}

	// Suggest missing imports at the end
	if hasValidation && !hasValidationImport {
		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: SeverityWarning,
			Source:   "buffalo-lsp",
			Message:  "buffalo.validate used but not imported. Add: import \"buffalo/validate/validate.proto\";",
		})
	}

	if hasPermissions && !hasPermissionsImport {
		diagnostics = append(diagnostics, Diagnostic{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
			Severity: SeverityWarning,
			Source:   "buffalo-lsp",
			Message:  "buffalo.permissions used but not imported",
		})
	}

	return diagnostics
}

// Complete returns completion items for a position.
func (a *ProtoAnalyzer) Complete(doc *Document, pos Position, ctx *CompletionContext) []CompletionItem {
	items := []CompletionItem{}

	line := doc.getLine(pos.Line)
	prefix := ""
	if pos.Character > 0 && pos.Character <= len(line) {
		prefix = line[:pos.Character]
	}

	// Determine completion context
	trimmed := strings.TrimSpace(prefix)

	// Top-level keywords
	if trimmed == "" || !strings.Contains(trimmed, " ") {
		items = append(items, topLevelCompletions()...)
	}

	// Inside message
	if isInsideMessage(doc, pos) {
		items = append(items, fieldTypeCompletions()...)
		items = append(items, fieldModifierCompletions()...)
	}

	// Inside service
	if isInsideService(doc, pos) {
		items = append(items, serviceCompletions()...)
	}

	// Buffalo annotations
	if strings.Contains(prefix, "[(") || strings.Contains(prefix, "buffalo") {
		items = append(items, buffaloAnnotationCompletions()...)
	}

	// After '=' for options
	if strings.HasSuffix(trimmed, "=") {
		items = append(items, optionValueCompletions()...)
	}

	// Import paths
	if strings.HasPrefix(trimmed, "import") && strings.Contains(line, "\"") {
		items = append(items, commonImportCompletions()...)
	}

	return items
}

// ResolveCompletion adds documentation to a completion item.
func (a *ProtoAnalyzer) ResolveCompletion(item CompletionItem) CompletionItem {
	// Add documentation based on completion kind
	switch item.Label {
	case "string", "int32", "int64", "uint32", "uint64", "bool", "bytes", "float", "double":
		item.Documentation = MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: getScalarTypeDoc(item.Label),
		}
	case "message":
		item.Documentation = MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: "Define a new message type.\n\n```protobuf\nmessage MyMessage {\n  string field = 1;\n}\n```",
		}
	case "service":
		item.Documentation = MarkupContent{
			Kind:  MarkupKindMarkdown,
			Value: "Define a gRPC service.\n\n```protobuf\nservice MyService {\n  rpc MyMethod(Request) returns (Response);\n}\n```",
		}
	}
	return item
}

// Hover returns hover information for a position.
func (a *ProtoAnalyzer) Hover(doc *Document, pos Position) *Hover {
	word := doc.getWordAtPosition(pos)
	if word == "" {
		return nil
	}

	// Check for scalar types
	if doc := getScalarTypeDoc(word); doc != "" {
		return &Hover{
			Contents: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: doc,
			},
		}
	}

	// Check for keywords
	if doc := getKeywordDoc(word); doc != "" {
		return &Hover{
			Contents: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: doc,
			},
		}
	}

	// Look for symbol definition in the document
	symbols := a.parseSymbols(doc)
	for _, sym := range symbols {
		if sym.Name == word {
			return &Hover{
				Contents: MarkupContent{
					Kind:  MarkupKindMarkdown,
					Value: formatSymbolHover(sym),
				},
			}
		}
		// Check children
		for _, child := range sym.Children {
			if child.Name == word {
				return &Hover{
					Contents: MarkupContent{
						Kind:  MarkupKindMarkdown,
						Value: formatSymbolHover(child),
					},
				}
			}
		}
	}

	return nil
}

// Definition returns the definition location for a position.
func (a *ProtoAnalyzer) Definition(doc *Document, pos Position) *Location {
	word := doc.getWordAtPosition(pos)
	if word == "" {
		return nil
	}

	// Look for symbol in current document
	symbols := a.parseSymbols(doc)
	for _, sym := range symbols {
		if sym.Name == word {
			return &Location{
				URI:   doc.URI,
				Range: sym.Range,
			}
		}
		for _, child := range sym.Children {
			if child.Name == word {
				return &Location{
					URI:   doc.URI,
					Range: child.Range,
				}
			}
		}
	}

	return nil
}

// References returns all references to a symbol.
func (a *ProtoAnalyzer) References(doc *Document, pos Position, docs []*Document, includeDeclaration bool) []Location {
	word := doc.getWordAtPosition(pos)
	if word == "" {
		return nil
	}

	locations := []Location{}

	for _, d := range docs {
		for lineNum, line := range d.Lines {
			// Simple word search
			idx := 0
			for {
				found := strings.Index(line[idx:], word)
				if found == -1 {
					break
				}
				pos := idx + found
				// Check word boundaries
				if (pos == 0 || !isWordChar(rune(line[pos-1]))) &&
					(pos+len(word) >= len(line) || !isWordChar(rune(line[pos+len(word)]))) {
					locations = append(locations, Location{
						URI: d.URI,
						Range: Range{
							Start: Position{Line: lineNum, Character: pos},
							End:   Position{Line: lineNum, Character: pos + len(word)},
						},
					})
				}
				idx = pos + len(word)
			}
		}
	}

	return locations
}

// DocumentSymbols returns document symbols.
func (a *ProtoAnalyzer) DocumentSymbols(doc *Document) []DocumentSymbol {
	symbols := a.parseSymbols(doc)
	return convertToDocumentSymbols(symbols)
}

// Format formats a document.
func (a *ProtoAnalyzer) Format(doc *Document, opts FormattingOptions) []TextEdit {
	edits := []TextEdit{}

	// Simple formatting: fix indentation
	indent := "  "
	if !opts.InsertSpaces {
		indent = "\t"
	} else if opts.TabSize > 0 {
		indent = strings.Repeat(" ", opts.TabSize)
	}

	currentIndent := 0
	for lineNum, line := range doc.Lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Decrease indent before closing brace
		if strings.HasPrefix(trimmed, "}") {
			currentIndent--
			if currentIndent < 0 {
				currentIndent = 0
			}
		}

		// Calculate expected indentation
		expectedIndent := strings.Repeat(indent, currentIndent)

		// Check if line needs reformatting
		if !strings.HasPrefix(line, expectedIndent) || strings.TrimPrefix(line, expectedIndent) != trimmed {
			newLine := expectedIndent + trimmed
			edits = append(edits, TextEdit{
				Range: Range{
					Start: Position{Line: lineNum, Character: 0},
					End:   Position{Line: lineNum, Character: len(line)},
				},
				NewText: newLine,
			})
		}

		// Increase indent after opening brace
		if strings.HasSuffix(trimmed, "{") {
			currentIndent++
		}
	}

	return edits
}

// CodeActions returns available code actions.
func (a *ProtoAnalyzer) CodeActions(doc *Document, rng Range, ctx CodeActionContext) []CodeAction {
	actions := []CodeAction{}

	// Quick fixes based on diagnostics
	for _, diag := range ctx.Diagnostics {
		if strings.Contains(diag.Message, "snake_case") {
			// Offer to convert to snake_case
			line := doc.getLine(diag.Range.Start.Line)
			if fieldPattern.MatchString(line) {
				matches := fieldPattern.FindStringSubmatch(line)
				if len(matches) > 3 {
					oldName := matches[3]
					newName := toSnakeCase(oldName)
					actions = append(actions, CodeAction{
						Title:       fmt.Sprintf("Convert '%s' to snake_case", oldName),
						Kind:        CodeActionKindQuickFix,
						Diagnostics: []Diagnostic{diag},
						Edit: &WorkspaceEdit{
							Changes: map[DocumentURI][]TextEdit{
								doc.URI: {{
									Range:   diag.Range,
									NewText: strings.Replace(line, oldName, newName, 1),
								}},
							},
						},
					})
				}
			}
		}

		if strings.Contains(diag.Message, "Missing syntax") {
			actions = append(actions, CodeAction{
				Title:       "Add syntax declaration",
				Kind:        CodeActionKindQuickFix,
				Diagnostics: []Diagnostic{diag},
				Edit: &WorkspaceEdit{
					Changes: map[DocumentURI][]TextEdit{
						doc.URI: {{
							Range: Range{
								Start: Position{Line: 0, Character: 0},
								End:   Position{Line: 0, Character: 0},
							},
							NewText: "syntax = \"proto3\";\n\n",
						}},
					},
				},
			})
		}
	}

	// Add validation annotation
	line := doc.getLine(rng.Start.Line)
	if fieldPattern.MatchString(line) && !strings.Contains(line, "[(") {
		actions = append(actions, CodeAction{
			Title: "Add validation annotation",
			Kind:  CodeActionKindRefactor,
			Edit: &WorkspaceEdit{
				Changes: map[DocumentURI][]TextEdit{
					doc.URI: {{
						Range: Range{
							Start: Position{Line: rng.Start.Line, Character: len(line) - 1},
							End:   Position{Line: rng.Start.Line, Character: len(line) - 1},
						},
						NewText: " [(buffalo.validate.rules).string = {}]",
					}},
				},
			},
		})
	}

	return actions
}

// PrepareRename checks if rename is possible.
func (a *ProtoAnalyzer) PrepareRename(doc *Document, pos Position) *PrepareRenameResult {
	word := doc.getWordAtPosition(pos)
	if word == "" {
		return nil
	}

	// Check if it's a renameable symbol
	line := doc.getLine(pos.Line)
	if messagePattern.MatchString(line) || servicePattern.MatchString(line) ||
		enumPattern.MatchString(line) || fieldPattern.MatchString(line) {

		// Find word position in line
		idx := strings.Index(line, word)
		if idx >= 0 {
			return &PrepareRenameResult{
				Range: Range{
					Start: Position{Line: pos.Line, Character: idx},
					End:   Position{Line: pos.Line, Character: idx + len(word)},
				},
				Placeholder: word,
			}
		}
	}

	return nil
}

// Rename renames a symbol.
func (a *ProtoAnalyzer) Rename(doc *Document, pos Position, newName string, docs []*Document) *WorkspaceEdit {
	refs := a.References(doc, pos, docs, true)
	if len(refs) == 0 {
		return nil
	}

	changes := make(map[DocumentURI][]TextEdit)

	for _, ref := range refs {
		changes[ref.URI] = append(changes[ref.URI], TextEdit{
			Range:   ref.Range,
			NewText: newName,
		})
	}

	return &WorkspaceEdit{Changes: changes}
}

// FoldingRanges returns folding ranges.
func (a *ProtoAnalyzer) FoldingRanges(doc *Document) []FoldingRange {
	ranges := []FoldingRange{}

	type stackItem struct {
		line int
		kind string
	}
	stack := []stackItem{}

	for lineNum, line := range doc.Lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasSuffix(trimmed, "{") {
			kind := "region"
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
				kind = "comment"
			}
			stack = append(stack, stackItem{lineNum, kind})
		}

		if strings.HasPrefix(trimmed, "}") && len(stack) > 0 {
			start := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			if lineNum > start.line {
				ranges = append(ranges, FoldingRange{
					StartLine: start.line,
					EndLine:   lineNum,
					Kind:      start.kind,
				})
			}
		}
	}

	return ranges
}

// SemanticTokens returns semantic tokens.
func (a *ProtoAnalyzer) SemanticTokens(doc *Document) *SemanticTokens {
	data := []int{}
	prevLine := 0
	prevChar := 0

	for lineNum, line := range doc.Lines {
		tokens := tokenizeLine(line)
		for _, tok := range tokens {
			// Delta encoding
			deltaLine := lineNum - prevLine
			deltaChar := tok.Start
			if deltaLine == 0 {
				deltaChar = tok.Start - prevChar
			}

			data = append(data, deltaLine, deltaChar, tok.Length, tok.Type, tok.Modifiers)

			prevLine = lineNum
			prevChar = tok.Start
		}
	}

	return &SemanticTokens{Data: data}
}

// parseSymbols parses symbols from a document.
func (a *ProtoAnalyzer) parseSymbols(doc *Document) []ProtoSymbol {
	symbols := []ProtoSymbol{}

	type parseContext struct {
		symbol    *ProtoSymbol
		startLine int
	}
	stack := []parseContext{}

	for lineNum, line := range doc.Lines {
		trimmed := strings.TrimSpace(line)

		// Message
		if matches := messagePattern.FindStringSubmatch(line); matches != nil {
			sym := ProtoSymbol{
				Name: matches[1],
				Kind: SymbolKindStruct,
				Range: Range{
					Start: Position{Line: lineNum, Character: strings.Index(line, matches[1])},
					End:   Position{Line: lineNum, Character: strings.Index(line, matches[1]) + len(matches[1])},
				},
			}
			if len(stack) > 0 {
				stack[len(stack)-1].symbol.Children = append(stack[len(stack)-1].symbol.Children, sym)
			} else {
				symbols = append(symbols, sym)
			}
			if strings.Contains(line, "{") {
				stack = append(stack, parseContext{&symbols[len(symbols)-1], lineNum})
			}
		}

		// Service
		if matches := servicePattern.FindStringSubmatch(line); matches != nil {
			sym := ProtoSymbol{
				Name: matches[1],
				Kind: SymbolKindInterface,
				Range: Range{
					Start: Position{Line: lineNum, Character: strings.Index(line, matches[1])},
					End:   Position{Line: lineNum, Character: strings.Index(line, matches[1]) + len(matches[1])},
				},
			}
			symbols = append(symbols, sym)
			if strings.Contains(line, "{") {
				stack = append(stack, parseContext{&symbols[len(symbols)-1], lineNum})
			}
		}

		// Enum
		if matches := enumPattern.FindStringSubmatch(line); matches != nil {
			sym := ProtoSymbol{
				Name: matches[1],
				Kind: SymbolKindEnum,
				Range: Range{
					Start: Position{Line: lineNum, Character: strings.Index(line, matches[1])},
					End:   Position{Line: lineNum, Character: strings.Index(line, matches[1]) + len(matches[1])},
				},
			}
			symbols = append(symbols, sym)
			if strings.Contains(line, "{") {
				stack = append(stack, parseContext{&symbols[len(symbols)-1], lineNum})
			}
		}

		// RPC
		if matches := rpcPattern.FindStringSubmatch(line); matches != nil && len(stack) > 0 {
			sym := ProtoSymbol{
				Name: matches[1],
				Kind: SymbolKindMethod,
				Range: Range{
					Start: Position{Line: lineNum, Character: strings.Index(line, matches[1])},
					End:   Position{Line: lineNum, Character: strings.Index(line, matches[1]) + len(matches[1])},
				},
			}
			stack[len(stack)-1].symbol.Children = append(stack[len(stack)-1].symbol.Children, sym)
		}

		// Field
		if matches := fieldPattern.FindStringSubmatch(line); matches != nil && len(stack) > 0 {
			sym := ProtoSymbol{
				Name: matches[3],
				Kind: SymbolKindField,
				Type: matches[2],
				Range: Range{
					Start: Position{Line: lineNum, Character: strings.Index(line, matches[3])},
					End:   Position{Line: lineNum, Character: strings.Index(line, matches[3]) + len(matches[3])},
				},
			}
			stack[len(stack)-1].symbol.Children = append(stack[len(stack)-1].symbol.Children, sym)
		}

		// Closing brace
		if strings.HasPrefix(trimmed, "}") && len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
	}

	return symbols
}

// Helper functions

func lineRange(line int, content string) Range {
	return Range{
		Start: Position{Line: line, Character: 0},
		End:   Position{Line: line, Character: len(content)},
	}
}

func isSnakeCase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return false
		}
	}
	return true
}

func isPascalCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= 'A' && s[0] <= 'Z'
}

func isUpperSnakeCase(s string) bool {
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return false
		}
	}
	return true
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(r + 32) // lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func isInsideMessage(doc *Document, pos Position) bool {
	braceCount := 0
	inMessage := false
	for i := 0; i <= pos.Line && i < len(doc.Lines); i++ {
		line := doc.Lines[i]
		if messagePattern.MatchString(line) {
			inMessage = true
		}
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
	}
	return inMessage && braceCount > 0
}

func isInsideService(doc *Document, pos Position) bool {
	braceCount := 0
	inService := false
	for i := 0; i <= pos.Line && i < len(doc.Lines); i++ {
		line := doc.Lines[i]
		if servicePattern.MatchString(line) {
			inService = true
		}
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
	}
	return inService && braceCount > 0
}

func convertToDocumentSymbols(symbols []ProtoSymbol) []DocumentSymbol {
	result := make([]DocumentSymbol, len(symbols))
	for i, sym := range symbols {
		result[i] = DocumentSymbol{
			Name:           sym.Name,
			Kind:           sym.Kind,
			Range:          sym.Range,
			SelectionRange: sym.Range,
			Children:       convertToDocumentSymbols(sym.Children),
		}
	}
	return result
}

func formatSymbolHover(sym ProtoSymbol) string {
	kindName := "symbol"
	switch sym.Kind {
	case SymbolKindStruct:
		kindName = "message"
	case SymbolKindInterface:
		kindName = "service"
	case SymbolKindEnum:
		kindName = "enum"
	case SymbolKindMethod:
		kindName = "rpc"
	case SymbolKindField:
		kindName = "field"
	}

	result := fmt.Sprintf("**%s** `%s`", kindName, sym.Name)
	if sym.Type != "" {
		result += fmt.Sprintf("\n\nType: `%s`", sym.Type)
	}
	if sym.Documentation != "" {
		result += "\n\n" + sym.Documentation
	}
	return result
}

type semanticToken struct {
	Start     int
	Length    int
	Type      int
	Modifiers int
}

func tokenizeLine(line string) []semanticToken {
	tokens := []semanticToken{}

	// Keywords
	keywords := []string{"syntax", "package", "import", "option", "message", "service", "enum", "rpc", "returns", "repeated", "optional", "required", "oneof", "map", "reserved", "extensions", "extend", "public", "weak"}
	for _, kw := range keywords {
		idx := 0
		for {
			found := strings.Index(line[idx:], kw)
			if found == -1 {
				break
			}
			pos := idx + found
			if (pos == 0 || !isWordChar(rune(line[pos-1]))) &&
				(pos+len(kw) >= len(line) || !isWordChar(rune(line[pos+len(kw)]))) {
				tokens = append(tokens, semanticToken{
					Start:  pos,
					Length: len(kw),
					Type:   13, // keyword
				})
			}
			idx = pos + len(kw)
		}
	}

	return tokens
}

// deduplicateDiagnostics removes diagnostics that have the same line and severity
// when a more specific (coded) diagnostic exists for the same issue.
func deduplicateDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	type diagKey struct {
		line     int
		severity DiagnosticSeverity
	}

	// Group by line + severity
	seen := make(map[diagKey][]int) // key -> indices
	for i, d := range diagnostics {
		key := diagKey{line: d.Range.Start.Line, severity: d.Severity}
		seen[key] = append(seen[key], i)
	}

	// For groups with multiple diagnostics on the same line+severity,
	// prefer the one with a diagnostic code (from SyntaxDiagnostics)
	remove := make(map[int]bool)
	for _, indices := range seen {
		if len(indices) <= 1 {
			continue
		}

		// Check if there are both coded and uncoded diagnostics
		hasCodedDiag := false
		for _, idx := range indices {
			if diagnostics[idx].Code != nil && diagnostics[idx].Code != "" {
				hasCodedDiag = true
				break
			}
		}

		// If we have coded diagnostics, remove uncoded ones on the same line
		if hasCodedDiag {
			for _, idx := range indices {
				if diagnostics[idx].Code == nil || diagnostics[idx].Code == "" {
					remove[idx] = true
				}
			}
		}
	}

	if len(remove) == 0 {
		return diagnostics
	}

	result := make([]Diagnostic, 0, len(diagnostics)-len(remove))
	for i, d := range diagnostics {
		if !remove[i] {
			result = append(result, d)
		}
	}
	return result
}
