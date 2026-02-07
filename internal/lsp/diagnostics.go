package lsp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DiagnosticCode constants for categorized diagnostics.
const (
	DiagCodeMissingSyntax        = "BUF001"
	DiagCodeInvalidSyntax        = "BUF002"
	DiagCodeMissingPackage       = "BUF003"
	DiagCodeMismatchedBraces     = "BUF004"
	DiagCodeInvalidFieldNumber   = "BUF005"
	DiagCodeReservedFieldNumber  = "BUF006"
	DiagCodeDuplicateFieldNumber = "BUF007"
	DiagCodeDuplicateFieldName   = "BUF008"
	DiagCodeNamingConvention     = "BUF009"
	DiagCodeMissingSemicolon     = "BUF010"
	DiagCodeInvalidType          = "BUF011"
	DiagCodeInvalidImport        = "BUF012"
	DiagCodeDuplicateImport      = "BUF013"
	DiagCodeInvalidOption        = "BUF014"
	DiagCodeInvalidRPC           = "BUF015"
	DiagCodeInvalidEnum          = "BUF016"
	DiagCodeInvalidMapKey        = "BUF017"
	DiagCodeUnterminatedString   = "BUF018"
	DiagCodeInvalidSyntaxGeneral = "BUF019"
	DiagCodeEmptyMessage         = "BUF020"
	DiagCodeTopLevelStatement    = "BUF021"
	DiagCodeDuplicatePackage     = "BUF022"
	DiagCodeDuplicateSyntax      = "BUF023"
	DiagCodeInvalidEnumDefault   = "BUF024"
	DiagCodeReservedKeyword      = "BUF025"
)

// syntaxDiagnosticSource is the source identifier for syntax diagnostics.
const syntaxDiagnosticSource = "buffalo-lsp-syntax"

// protoScalarTypes contains all valid proto scalar types.
var protoScalarTypes = map[string]bool{
	"double": true, "float": true,
	"int32": true, "int64": true,
	"uint32": true, "uint64": true,
	"sint32": true, "sint64": true,
	"fixed32": true, "fixed64": true,
	"sfixed32": true, "sfixed64": true,
	"bool": true, "string": true, "bytes": true,
}

// protoReservedKeywords contains reserved proto keywords that cannot be used as identifiers.
var protoReservedKeywords = map[string]bool{
	"syntax": true, "import": true, "weak": true, "public": true,
	"package": true, "option": true, "inf": true, "repeated": true,
	"optional": true, "required": true, "group": true, "oneof": true,
	"map": true, "extensions": true, "to": true, "max": true,
	"reserved": true, "enum": true, "message": true, "extend": true,
	"service": true, "rpc": true, "stream": true, "returns": true,
	"true": true, "false": true,
}

// validMapKeyTypes contains valid types for map keys.
var validMapKeyTypes = map[string]bool{
	"int32": true, "int64": true,
	"uint32": true, "uint64": true,
	"sint32": true, "sint64": true,
	"fixed32": true, "fixed64": true,
	"sfixed32": true, "sfixed64": true,
	"bool": true, "string": true,
}

// Additional regex patterns for syntax diagnostics.
var (
	// Detects incomplete option statements.
	incompleteOptionPattern = regexp.MustCompile(`^\s*option\s+[^;]*$`)

	// Detects invalid field declarations.
	invalidFieldDeclPattern = regexp.MustCompile(`^\s*(repeated|optional|required)?\s*([a-zA-Z_][a-zA-Z0-9_.]*)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*$`)

	// Detects lines that look like statements but miss semicolons.
	statementNoSemicolon = regexp.MustCompile(`^\s*(syntax|package|import|option)\s+.+[^;{}\s]\s*$`)

	// Detects malformed RPC declarations.
	rpcFullPattern = regexp.MustCompile(`^\s*rpc\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(\s*(stream\s+)?([a-zA-Z_][a-zA-Z0-9_.]*)\s*\)\s*returns\s*\(\s*(stream\s+)?([a-zA-Z_][a-zA-Z0-9_.]*)\s*\)`)

	// Detects incomplete RPC (missing returns).
	rpcIncompletePattern = regexp.MustCompile(`^\s*rpc\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\)\s*;?\s*$`)

	// Detects syntax statement.
	syntaxFullPattern = regexp.MustCompile(`^\s*syntax\s*=\s*["']([^"']*)["']\s*;\s*$`)

	// Detects unterminated strings.
	unterminatedStringPattern = regexp.MustCompile(`["'][^"']*$`)

	// Detects map field declaration.
	mapFieldFullPattern = regexp.MustCompile(`^\s*map\s*<\s*([^,]+?)\s*,\s*([^>]+?)\s*>\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(\d+)\s*`)

	// Detects duplicate field number assignment (for multi-line scanning).
	fieldNumberPattern = regexp.MustCompile(`=\s*(\d+)\s*[;\[\s]`)
)

// SyntaxDiagnostics performs deep syntax analysis on a proto document.
// This complements the existing Analyze method with more detailed syntax checks.
func (a *ProtoAnalyzer) SyntaxDiagnostics(doc *Document) []Diagnostic {
	diagnostics := []Diagnostic{}

	if !strings.HasSuffix(string(doc.URI), ".proto") {
		return diagnostics
	}

	ctx := newSyntaxContext(doc)

	// Phase 1: Line-by-line syntax analysis
	a.analyzeLineSyntax(ctx)

	// Phase 2: Structural analysis
	a.analyzeStructure(ctx)

	// Phase 3: Cross-reference analysis
	a.analyzeCrossReferences(ctx)

	return ctx.diagnostics
}

// syntaxContext holds state during syntax analysis.
type syntaxContext struct {
	doc            *Document
	diagnostics    []Diagnostic
	inBlockComment bool

	// Tracking declarations
	syntaxCount  int
	packageCount int
	syntaxLine   int
	packageLine  int

	// Import tracking
	imports     map[string]int // import path -> line number
	importLines []int

	// Scope tracking
	scopeStack   []scopeEntry
	currentScope scopeKind

	// Field tracking per message scope
	fieldNumbers map[string]map[int]int    // scope -> field number -> line
	fieldNames   map[string]map[string]int // scope -> field name -> line

	// Enum tracking
	enumValues map[string]map[int]int    // scope -> enum value -> line
	enumNames  map[string]map[string]int // scope -> enum name -> line

	// Multi-line option value tracking (e.g., option ... = { ... };)
	optionValueDepth int
}

type scopeKind int

const (
	scopeTopLevel scopeKind = iota
	scopeMessage
	scopeEnum
	scopeService
	scopeOneof
	scopeExtend
	scopeRPCBody
)

type scopeEntry struct {
	kind     scopeKind
	name     string
	line     int
	bracePos int
}

func newSyntaxContext(doc *Document) *syntaxContext {
	return &syntaxContext{
		doc:          doc,
		diagnostics:  []Diagnostic{},
		imports:      make(map[string]int),
		fieldNumbers: make(map[string]map[int]int),
		fieldNames:   make(map[string]map[string]int),
		enumValues:   make(map[string]map[int]int),
		enumNames:    make(map[string]map[string]int),
		currentScope: scopeTopLevel,
	}
}

func (ctx *syntaxContext) addDiagnostic(lineNum int, line string, severity DiagnosticSeverity, code, message string) {
	ctx.diagnostics = append(ctx.diagnostics, Diagnostic{
		Range:    lineRange(lineNum, line),
		Severity: severity,
		Code:     code,
		Source:   syntaxDiagnosticSource,
		Message:  message,
	})
}

func (ctx *syntaxContext) addDiagnosticWithRange(r Range, severity DiagnosticSeverity, code, message string) {
	ctx.diagnostics = append(ctx.diagnostics, Diagnostic{
		Range:    r,
		Severity: severity,
		Code:     code,
		Source:   syntaxDiagnosticSource,
		Message:  message,
	})
}

func (ctx *syntaxContext) scopePath() string {
	parts := []string{}
	for _, s := range ctx.scopeStack {
		parts = append(parts, s.name)
	}
	return strings.Join(parts, ".")
}

func (ctx *syntaxContext) pushScope(kind scopeKind, name string, line int) {
	ctx.scopeStack = append(ctx.scopeStack, scopeEntry{
		kind: kind,
		name: name,
		line: line,
	})
	ctx.currentScope = kind

	// Initialize field/enum tracking for new scope
	path := ctx.scopePath()
	if kind == scopeMessage || kind == scopeOneof {
		ctx.fieldNumbers[path] = make(map[int]int)
		ctx.fieldNames[path] = make(map[string]int)
	}
	if kind == scopeEnum {
		ctx.enumValues[path] = make(map[int]int)
		ctx.enumNames[path] = make(map[string]int)
	}
}

func (ctx *syntaxContext) popScope() {
	if len(ctx.scopeStack) > 0 {
		ctx.scopeStack = ctx.scopeStack[:len(ctx.scopeStack)-1]
	}
	if len(ctx.scopeStack) > 0 {
		ctx.currentScope = ctx.scopeStack[len(ctx.scopeStack)-1].kind
	} else {
		ctx.currentScope = scopeTopLevel
	}
}

// analyzeLineSyntax performs line-by-line syntax checking.
func (a *ProtoAnalyzer) analyzeLineSyntax(ctx *syntaxContext) {
	for lineNum, line := range ctx.doc.Lines {
		// Handle block comments
		if ctx.inBlockComment {
			if blockCommentEnd.MatchString(line) {
				ctx.inBlockComment = false
			}
			continue
		}
		if blockCommentStart.MatchString(line) && !blockCommentEnd.MatchString(line) {
			ctx.inBlockComment = true
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Skip empty lines and single-line comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Strip trailing inline comments for analysis
		cleanLine := stripInlineComment(trimmed)

		// Check for unterminated strings
		a.checkUnterminatedStrings(ctx, lineNum, line, cleanLine)

		// Syntax declaration checks
		a.checkSyntaxDeclaration(ctx, lineNum, line, cleanLine)

		// Package declaration checks
		a.checkPackageDeclaration(ctx, lineNum, line, cleanLine)

		// Import checks
		a.checkImportSyntax(ctx, lineNum, line, cleanLine)

		// Track multi-line option/annotation values (e.g., option ... = { ... }; or field = N [(rule) = { ... }];)
		if ctx.optionValueDepth > 0 {
			// We are inside a multi-line value literal — count braces and skip checks
			opens, closes := countBracesOutsideStrings(cleanLine)
			ctx.optionValueDepth += opens - closes
			if ctx.optionValueDepth < 0 {
				ctx.optionValueDepth = 0
			}
			continue
		}

		// Detect start of a multi-line value on this line.
		// This covers both:
		//   option (foo) = { ... };
		//   string name = 1 [(buffalo.validate.rules).string = { ... }];
		{
			opens, closes := countBracesOutsideStrings(cleanLine)
			delta := opens - closes
			// Lines that open a scope (message/service/enum/oneof/rpc body) are NOT annotation values.
			// They are handled by trackScopes below. We only track unclosed braces
			// for option statements and field annotations.
			isStructuralDecl := messagePattern.MatchString(cleanLine) ||
				servicePattern.MatchString(cleanLine) ||
				enumPattern.MatchString(cleanLine) ||
				oneofPattern.MatchString(cleanLine) ||
				strings.HasPrefix(cleanLine, "extend") ||
				(ctx.currentScope == scopeService && rpcPattern.MatchString(cleanLine))

			if delta > 0 && !isStructuralDecl {
				// Entering a multi-line option/annotation value
				ctx.optionValueDepth = delta
				continue
			}
		}

		// Save scope before tracking braces (declaration lines belong to the outer scope)
		scopeBeforeTracking := ctx.currentScope

		// Track scopes (braces)
		a.trackScopes(ctx, lineNum, line, cleanLine)

		// Context-specific checks use the scope BEFORE brace tracking
		// so that "enum Foo {" is checked in the outer scope, not inside the enum
		switch scopeBeforeTracking {
		case scopeTopLevel:
			a.checkTopLevelSyntax(ctx, lineNum, line, cleanLine)
		case scopeMessage, scopeOneof:
			a.checkMessageBodySyntax(ctx, lineNum, line, cleanLine)
		case scopeEnum:
			a.checkEnumBodySyntax(ctx, lineNum, line, cleanLine)
		case scopeService:
			a.checkServiceBodySyntax(ctx, lineNum, line, cleanLine)
		case scopeRPCBody:
			a.checkRPCBodySyntax(ctx, lineNum, line, cleanLine)
		}
	}

	// Check for unterminated block comment
	if ctx.inBlockComment {
		lastLine := len(ctx.doc.Lines) - 1
		if lastLine < 0 {
			lastLine = 0
		}
		ctx.addDiagnostic(lastLine, ctx.doc.getLine(lastLine), SeverityError,
			DiagCodeInvalidSyntaxGeneral, "Unterminated block comment")
	}
}

// checkUnterminatedStrings checks for unterminated string literals on a line.
func (a *ProtoAnalyzer) checkUnterminatedStrings(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	// Count unescaped quotes
	doubleQuotes := 0
	singleQuotes := 0
	escaped := false
	for _, ch := range cleanLine {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			doubleQuotes++
		}
		if ch == '\'' {
			singleQuotes++
		}
	}
	if doubleQuotes%2 != 0 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeUnterminatedString, "Unterminated string literal (unmatched '\"')")
	}
	if singleQuotes%2 != 0 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeUnterminatedString, "Unterminated string literal (unmatched \"'\")")
	}
}

// checkSyntaxDeclaration validates the syntax declaration.
func (a *ProtoAnalyzer) checkSyntaxDeclaration(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	if !strings.HasPrefix(cleanLine, "syntax") {
		return
	}

	ctx.syntaxCount++
	ctx.syntaxLine = lineNum

	if ctx.syntaxCount > 1 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeDuplicateSyntax, "Duplicate syntax declaration; only one syntax statement is allowed per file")
		return
	}

	if len(ctx.scopeStack) > 0 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidSyntaxGeneral, "Syntax declaration must be at the top level")
		return
	}

	// Validate syntax statement format
	if !syntaxFullPattern.MatchString(cleanLine) {
		if !strings.Contains(cleanLine, "=") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeInvalidSyntax, "Invalid syntax declaration: expected 'syntax = \"proto3\";'")
		} else if !strings.HasSuffix(cleanLine, ";") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeMissingSemicolon, "Missing semicolon at end of syntax declaration")
		} else {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeInvalidSyntax, "Invalid syntax declaration format; expected 'syntax = \"proto2\";' or 'syntax = \"proto3\";'")
		}
		return
	}

	matches := syntaxFullPattern.FindStringSubmatch(cleanLine)
	if len(matches) > 1 && matches[1] != "proto3" && matches[1] != "proto2" {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidSyntax,
			fmt.Sprintf("Invalid syntax version '%s'; expected 'proto2' or 'proto3'", matches[1]))
	}
}

// checkPackageDeclaration validates the package declaration.
func (a *ProtoAnalyzer) checkPackageDeclaration(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	if !strings.HasPrefix(cleanLine, "package") {
		return
	}

	ctx.packageCount++
	ctx.packageLine = lineNum

	if ctx.packageCount > 1 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeDuplicatePackage, "Duplicate package declaration; only one package statement is allowed per file")
		return
	}

	if len(ctx.scopeStack) > 0 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidSyntaxGeneral, "Package declaration must be at the top level")
		return
	}

	if !strings.HasSuffix(cleanLine, ";") {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeMissingSemicolon, "Missing semicolon at end of package declaration")
		return
	}

	// Validate package name
	pkgMatch := packagePattern.FindStringSubmatch(cleanLine)
	if pkgMatch == nil {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidSyntaxGeneral, "Invalid package declaration; package name must be a valid identifier (e.g., 'package my.package;')")
	}
}

// checkImportSyntax validates import statements.
func (a *ProtoAnalyzer) checkImportSyntax(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	if !strings.HasPrefix(cleanLine, "import") {
		return
	}

	if len(ctx.scopeStack) > 0 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidImport, "Import statements must be at the top level")
		return
	}

	if !strings.HasSuffix(cleanLine, ";") {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeMissingSemicolon, "Missing semicolon at end of import statement")
		return
	}

	matches := importPattern.FindStringSubmatch(cleanLine)
	if matches == nil {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidImport, "Invalid import statement; expected 'import \"path/to/file.proto\";'")
		return
	}

	importPath := matches[2]

	// Check for empty import path
	if importPath == "" {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidImport, "Import path cannot be empty")
		return
	}

	// Check for .proto extension
	if !strings.HasSuffix(importPath, ".proto") {
		ctx.addDiagnostic(lineNum, line, SeverityWarning,
			DiagCodeInvalidImport, "Import path should end with '.proto' extension")
	}

	// Check for duplicate imports
	if prevLine, exists := ctx.imports[importPath]; exists {
		ctx.addDiagnostic(lineNum, line, SeverityWarning,
			DiagCodeDuplicateImport,
			fmt.Sprintf("Duplicate import '%s' (first imported at line %d)", importPath, prevLine+1))
	} else {
		ctx.imports[importPath] = lineNum
	}

	ctx.importLines = append(ctx.importLines, lineNum)
}

// trackScopes tracks opening and closing braces to maintain scope context.
func (a *ProtoAnalyzer) trackScopes(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	// Count braces excluding those inside strings
	opens, closes := countBracesOutsideStrings(cleanLine)

	// Process closing braces first (for lines like "} ... {" transitions)
	// But handle opens before closes if they appear on the same declaration line
	if opens > 0 {
		// Determine what kind of scope this opens
		matched := false
		if messagePattern.MatchString(cleanLine) {
			matches := messagePattern.FindStringSubmatch(cleanLine)
			if len(matches) > 1 {
				ctx.pushScope(scopeMessage, matches[1], lineNum)
				opens--
				matched = true
			}
		}
		if !matched && servicePattern.MatchString(cleanLine) {
			matches := servicePattern.FindStringSubmatch(cleanLine)
			if len(matches) > 1 {
				ctx.pushScope(scopeService, matches[1], lineNum)
				opens--
				matched = true
			}
		}
		if !matched && enumPattern.MatchString(cleanLine) {
			matches := enumPattern.FindStringSubmatch(cleanLine)
			if len(matches) > 1 {
				ctx.pushScope(scopeEnum, matches[1], lineNum)
				opens--
				matched = true
			}
		}
		if !matched && oneofPattern.MatchString(cleanLine) {
			matches := oneofPattern.FindStringSubmatch(cleanLine)
			if len(matches) > 1 {
				ctx.pushScope(scopeOneof, matches[1], lineNum)
				opens--
				matched = true
			}
		}
		if !matched && strings.HasPrefix(cleanLine, "extend") {
			ctx.pushScope(scopeExtend, "extend", lineNum)
			opens--
			matched = true
		}
		if !matched && ctx.currentScope == scopeService && rpcPattern.MatchString(cleanLine) {
			// RPC body: rpc Foo(Req) returns (Resp) {
			ctx.pushScope(scopeRPCBody, "rpc", lineNum)
			opens--
			matched = true
		}

		// Handle any remaining unmatched opens
		for i := 0; i < opens; i++ {
			ctx.pushScope(ctx.currentScope, "", lineNum)
		}
	}

	for i := 0; i < closes; i++ {
		if len(ctx.scopeStack) == 0 {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeMismatchedBraces, "Unexpected closing brace '}' without matching opening brace")
		} else {
			ctx.popScope()
		}
	}
}

// checkTopLevelSyntax validates statements at the top level.
func (a *ProtoAnalyzer) checkTopLevelSyntax(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	// Already handled by other specific checks
	if strings.HasPrefix(cleanLine, "syntax") ||
		strings.HasPrefix(cleanLine, "package") ||
		strings.HasPrefix(cleanLine, "import") {
		return
	}

	// Valid top-level constructs
	if messagePattern.MatchString(cleanLine) ||
		servicePattern.MatchString(cleanLine) ||
		enumPattern.MatchString(cleanLine) ||
		strings.HasPrefix(cleanLine, "extend") ||
		strings.HasPrefix(cleanLine, "option") ||
		strings.HasPrefix(cleanLine, "}") {
		// Check option statements for semicolons
		if strings.HasPrefix(cleanLine, "option") && !strings.HasSuffix(cleanLine, ";") && !strings.HasSuffix(cleanLine, "{") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeMissingSemicolon, "Missing semicolon at end of option statement")
		}
		return
	}

	// Detect common invalid top-level statements
	if fieldPattern.MatchString(cleanLine) {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeTopLevelStatement, "Field declaration not allowed at top level; must be inside a message or extend block")
		return
	}

	if rpcPattern.MatchString(cleanLine) {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeTopLevelStatement, "RPC declaration not allowed at top level; must be inside a service block")
		return
	}

	if enumValuePattern.MatchString(cleanLine) {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeTopLevelStatement, "Enum value not allowed at top level; must be inside an enum block")
		return
	}

	// Catch-all: unrecognized statement at top level
	if !strings.HasPrefix(cleanLine, "//") && !strings.HasPrefix(cleanLine, "/*") && cleanLine != "" {
		// Only warn about truly unrecognized statements
		if !strings.Contains(cleanLine, "{") && !strings.Contains(cleanLine, "}") {
			ctx.addDiagnostic(lineNum, line, SeverityWarning,
				DiagCodeTopLevelStatement,
				fmt.Sprintf("Unexpected statement at top level: '%s'", truncateStr(cleanLine, 40)))
		}
	}
}

// checkMessageBodySyntax validates statements inside a message body.
func (a *ProtoAnalyzer) checkMessageBodySyntax(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	// Closing brace
	if strings.HasPrefix(cleanLine, "}") {
		return
	}

	// Nested types are allowed
	if messagePattern.MatchString(cleanLine) ||
		enumPattern.MatchString(cleanLine) ||
		oneofPattern.MatchString(cleanLine) ||
		strings.HasPrefix(cleanLine, "extend") ||
		strings.HasPrefix(cleanLine, "reserved") ||
		strings.HasPrefix(cleanLine, "extensions") {
		return
	}

	// Option statements
	if strings.HasPrefix(cleanLine, "option") {
		if !strings.HasSuffix(cleanLine, ";") && !strings.HasSuffix(cleanLine, "{") && !strings.Contains(cleanLine, "}") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeMissingSemicolon, "Missing semicolon at end of option statement")
		}
		return
	}

	// Map field check
	if mapFieldFullPattern.MatchString(cleanLine) {
		matches := mapFieldFullPattern.FindStringSubmatch(cleanLine)
		if len(matches) > 4 {
			keyType := strings.TrimSpace(matches[1])
			fieldName := matches[3]
			fieldNumStr := matches[4]

			// Validate map key type
			if !validMapKeyTypes[keyType] {
				ctx.addDiagnostic(lineNum, line, SeverityError,
					DiagCodeInvalidMapKey,
					fmt.Sprintf("Invalid map key type '%s'; map keys must be an integral or string type", keyType))
			}

			// Track field number
			a.trackFieldNumber(ctx, lineNum, line, fieldName, fieldNumStr)
		}
		return
	}

	// Regular field check
	if fieldPattern.MatchString(cleanLine) {
		matches := fieldPattern.FindStringSubmatch(cleanLine)
		if len(matches) > 4 {
			fieldType := matches[2]
			fieldName := matches[3]
			fieldNumStr := matches[4]

			// Check if field type is a reserved keyword used incorrectly
			if protoReservedKeywords[fieldName] {
				ctx.addDiagnostic(lineNum, line, SeverityError,
					DiagCodeReservedKeyword,
					fmt.Sprintf("'%s' is a reserved keyword and cannot be used as a field name", fieldName))
			}

			// Validate field type is not a keyword that shouldn't be a type
			invalidTypes := map[string]bool{
				"syntax": true, "import": true, "package": true,
				"option": true, "service": true, "rpc": true,
				"returns": true, "stream": true,
			}
			if invalidTypes[fieldType] {
				ctx.addDiagnostic(lineNum, line, SeverityError,
					DiagCodeInvalidType,
					fmt.Sprintf("'%s' is not a valid field type", fieldType))
			}

			// Track field number and name
			a.trackFieldNumber(ctx, lineNum, line, fieldName, fieldNumStr)

			// Check missing semicolon
			if !strings.Contains(cleanLine, ";") {
				ctx.addDiagnostic(lineNum, line, SeverityError,
					DiagCodeMissingSemicolon, "Missing semicolon at end of field declaration")
			}
		}
		return
	}

	// Detect malformed field declarations
	if invalidFieldDeclPattern.MatchString(cleanLine) {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidSyntaxGeneral,
			"Incomplete field declaration: missing field number assignment (e.g., '= 1;')")
		return
	}
}

// checkEnumBodySyntax validates statements inside an enum body.
func (a *ProtoAnalyzer) checkEnumBodySyntax(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	if strings.HasPrefix(cleanLine, "}") {
		return
	}

	// Option statements in enum
	if strings.HasPrefix(cleanLine, "option") {
		if !strings.HasSuffix(cleanLine, ";") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeMissingSemicolon, "Missing semicolon at end of option statement")
		}
		return
	}

	// Reserved statements
	if strings.HasPrefix(cleanLine, "reserved") {
		if !strings.HasSuffix(cleanLine, ";") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeMissingSemicolon, "Missing semicolon at end of reserved statement")
		}
		return
	}

	// Enum values
	if enumValuePattern.MatchString(cleanLine) {
		matches := enumValuePattern.FindStringSubmatch(cleanLine)
		if len(matches) > 2 {
			valueName := matches[1]
			valueNumStr := matches[2]
			valueNum, err := strconv.Atoi(valueNumStr)

			scopePath := ctx.scopePath()

			if err == nil {
				// Check for first enum value must be 0
				if enumVals, ok := ctx.enumValues[scopePath]; ok {
					if len(enumVals) == 0 && valueNum != 0 {
						// Only for proto3
						ctx.addDiagnostic(lineNum, line, SeverityWarning,
							DiagCodeInvalidEnumDefault,
							"First enum value should be 0 in proto3 (default value)")
					}

					// Check for duplicate enum values
					if prevLine, exists := enumVals[valueNum]; exists {
						ctx.addDiagnostic(lineNum, line, SeverityError,
							DiagCodeDuplicateFieldNumber,
							fmt.Sprintf("Duplicate enum value %d (previously defined at line %d)", valueNum, prevLine+1))
					} else {
						enumVals[valueNum] = lineNum
					}
				}

				// Check for duplicate enum names
				if enumNames, ok := ctx.enumNames[scopePath]; ok {
					if prevLine, exists := enumNames[valueName]; exists {
						ctx.addDiagnostic(lineNum, line, SeverityError,
							DiagCodeDuplicateFieldName,
							fmt.Sprintf("Duplicate enum value name '%s' (previously defined at line %d)", valueName, prevLine+1))
					} else {
						enumNames[valueName] = lineNum
					}
				}
			}

			// Check semicolon
			if !strings.Contains(cleanLine, ";") {
				ctx.addDiagnostic(lineNum, line, SeverityError,
					DiagCodeMissingSemicolon, "Missing semicolon at end of enum value declaration")
			}
		}
		return
	}

	// If none of the above, it's invalid syntax in enum
	if cleanLine != "" {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidSyntaxGeneral,
			fmt.Sprintf("Invalid statement inside enum: '%s'; expected enum value declaration (e.g., 'VALUE_NAME = 0;')", truncateStr(cleanLine, 40)))
	}
}

// checkServiceBodySyntax validates statements inside a service body.
func (a *ProtoAnalyzer) checkServiceBodySyntax(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	if strings.HasPrefix(cleanLine, "}") {
		return
	}

	// Option statements in service
	if strings.HasPrefix(cleanLine, "option") {
		if !strings.HasSuffix(cleanLine, ";") && !strings.Contains(cleanLine, "{") && !strings.Contains(cleanLine, "}") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeMissingSemicolon, "Missing semicolon at end of option statement")
		}
		return
	}

	// RPC declarations
	if rpcPattern.MatchString(cleanLine) {
		// Check for complete RPC with returns
		if !strings.Contains(cleanLine, "returns") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeInvalidRPC, "Incomplete RPC declaration: missing 'returns' clause")
			return
		}

		// Check for proper RPC format
		if !rpcFullPattern.MatchString(cleanLine) && !rpcIncompletePattern.MatchString(cleanLine) {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeInvalidRPC,
				"Invalid RPC declaration format; expected 'rpc MethodName(RequestType) returns (ResponseType);'")
			return
		}

		// Check RPC ending (either ; or {)
		if !strings.HasSuffix(cleanLine, ";") && !strings.HasSuffix(cleanLine, "{") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeMissingSemicolon, "RPC declaration must end with ';' or '{'")
		}
		return
	}

	// Field declarations are not allowed in service
	if fieldPattern.MatchString(cleanLine) {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidSyntaxGeneral,
			"Field declarations are not allowed inside a service; use 'rpc' to define methods")
		return
	}

	// Detect unrecognized statements
	if cleanLine != "" && !strings.HasPrefix(cleanLine, "//") {
		ctx.addDiagnostic(lineNum, line, SeverityWarning,
			DiagCodeInvalidSyntaxGeneral,
			fmt.Sprintf("Unexpected statement inside service: '%s'", truncateStr(cleanLine, 40)))
	}
}

// checkRPCBodySyntax validates statements inside an RPC body.
func (a *ProtoAnalyzer) checkRPCBodySyntax(ctx *syntaxContext, lineNum int, line, cleanLine string) {
	if strings.HasPrefix(cleanLine, "}") {
		return
	}

	// Only option statements are valid inside RPC body
	if strings.HasPrefix(cleanLine, "option") {
		if !strings.HasSuffix(cleanLine, ";") && !strings.Contains(cleanLine, "{") && !strings.Contains(cleanLine, "}") {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeMissingSemicolon, "Missing semicolon at end of option statement")
		}
		return
	}

	if cleanLine != "" && !strings.HasPrefix(cleanLine, "//") {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidSyntaxGeneral,
			"Only option statements are allowed inside an RPC body")
	}
}

// trackFieldNumber tracks field numbers for duplicate detection.
func (a *ProtoAnalyzer) trackFieldNumber(ctx *syntaxContext, lineNum int, line, fieldName, fieldNumStr string) {
	fieldNum, err := strconv.Atoi(fieldNumStr)
	if err != nil {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidFieldNumber,
			fmt.Sprintf("Invalid field number '%s': must be a positive integer", fieldNumStr))
		return
	}

	// Validate field number range
	if fieldNum <= 0 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidFieldNumber, "Field number must be a positive integer (>= 1)")
	} else if fieldNum >= 19000 && fieldNum <= 19999 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeReservedFieldNumber,
			fmt.Sprintf("Field number %d is in the reserved range 19000-19999 (reserved for protobuf implementation)", fieldNum))
	} else if fieldNum > 536870911 {
		ctx.addDiagnostic(lineNum, line, SeverityError,
			DiagCodeInvalidFieldNumber,
			fmt.Sprintf("Field number %d exceeds maximum allowed value (536870911)", fieldNum))
	}

	// Check for duplicate field number
	scopePath := ctx.scopePath()
	if fieldNums, ok := ctx.fieldNumbers[scopePath]; ok {
		if prevLine, exists := fieldNums[fieldNum]; exists {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeDuplicateFieldNumber,
				fmt.Sprintf("Duplicate field number %d (previously used at line %d)", fieldNum, prevLine+1))
		} else {
			fieldNums[fieldNum] = lineNum
		}
	}

	// Check for duplicate field name
	if fieldNameMap, ok := ctx.fieldNames[scopePath]; ok {
		if prevLine, exists := fieldNameMap[fieldName]; exists {
			ctx.addDiagnostic(lineNum, line, SeverityError,
				DiagCodeDuplicateFieldName,
				fmt.Sprintf("Duplicate field name '%s' (previously defined at line %d)", fieldName, prevLine+1))
		} else {
			fieldNameMap[fieldName] = lineNum
		}
	}
}

// analyzeStructure performs structural analysis after line-by-line scan.
func (a *ProtoAnalyzer) analyzeStructure(ctx *syntaxContext) {
	// Check unclosed scopes
	for len(ctx.scopeStack) > 0 {
		scope := ctx.scopeStack[len(ctx.scopeStack)-1]
		scopeName := scope.name
		if scopeName == "" {
			scopeName = "block"
		}
		ctx.addDiagnostic(scope.line, ctx.doc.getLine(scope.line), SeverityError,
			DiagCodeMismatchedBraces,
			fmt.Sprintf("Unclosed '%s' block starting at this line (missing closing '}')", scopeName))
		ctx.popScope()
	}

	// Check missing syntax declaration
	if ctx.syntaxCount == 0 {
		ctx.addDiagnosticWithRange(Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 0},
		}, SeverityWarning, DiagCodeMissingSyntax,
			"Missing syntax declaration; add 'syntax = \"proto3\";' at the beginning of the file")
	}

	// Check missing package declaration
	if ctx.packageCount == 0 {
		ctx.addDiagnosticWithRange(Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 0},
		}, SeverityWarning, DiagCodeMissingPackage,
			"Missing package declaration; consider adding 'package <name>;'")
	}

	// Check syntax before package order
	if ctx.syntaxCount > 0 && ctx.packageCount > 0 && ctx.syntaxLine > ctx.packageLine {
		ctx.addDiagnostic(ctx.syntaxLine, ctx.doc.getLine(ctx.syntaxLine), SeverityWarning,
			DiagCodeInvalidSyntaxGeneral,
			"Syntax declaration should appear before the package declaration")
	}

	// Check imports appear after syntax/package but before other declarations
	if len(ctx.importLines) > 0 && ctx.syntaxCount > 0 {
		for _, importLine := range ctx.importLines {
			if importLine < ctx.syntaxLine {
				ctx.addDiagnostic(importLine, ctx.doc.getLine(importLine), SeverityWarning,
					DiagCodeInvalidImport,
					"Import statements should appear after the syntax declaration")
				break
			}
		}
	}
}

// analyzeCrossReferences performs cross-reference checks.
func (a *ProtoAnalyzer) analyzeCrossReferences(ctx *syntaxContext) {
	// Collect all defined type names
	definedTypes := make(map[string]bool)
	for _, line := range ctx.doc.Lines {
		if matches := messagePattern.FindStringSubmatch(line); matches != nil {
			definedTypes[matches[1]] = true
		}
		if matches := enumPattern.FindStringSubmatch(line); matches != nil {
			definedTypes[matches[1]] = true
		}
	}

	// Check RPC request/response types exist
	for lineNum, line := range ctx.doc.Lines {
		if rpcFullPattern.MatchString(line) {
			matches := rpcFullPattern.FindStringSubmatch(line)
			if len(matches) > 5 {
				reqType := matches[3]
				respType := matches[5]

				// Skip well-known types and qualified names
				if !strings.Contains(reqType, ".") && !protoScalarTypes[reqType] && !definedTypes[reqType] {
					ctx.addDiagnostic(lineNum, line, SeverityWarning,
						DiagCodeInvalidType,
						fmt.Sprintf("Request type '%s' is not defined in this file", reqType))
				}
				if !strings.Contains(respType, ".") && !protoScalarTypes[respType] && !definedTypes[respType] {
					ctx.addDiagnostic(lineNum, line, SeverityWarning,
						DiagCodeInvalidType,
						fmt.Sprintf("Response type '%s' is not defined in this file", respType))
				}
			}
		}
	}

	// Check field types reference existing types
	inScope := false
	for lineNum, line := range ctx.doc.Lines {
		if messagePattern.MatchString(line) || servicePattern.MatchString(line) || enumPattern.MatchString(line) {
			inScope = true
		}

		if inScope && fieldPattern.MatchString(line) {
			matches := fieldPattern.FindStringSubmatch(line)
			if len(matches) > 2 {
				fieldType := matches[2]
				// Skip scalar types, qualified names, and well-known types
				if !protoScalarTypes[fieldType] &&
					!strings.Contains(fieldType, ".") &&
					!definedTypes[fieldType] &&
					fieldType != "google" {
					ctx.addDiagnostic(lineNum, line, SeverityInformation,
						DiagCodeInvalidType,
						fmt.Sprintf("Type '%s' is not defined in this file; ensure it is imported or defined elsewhere", fieldType))
				}
			}
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "}" {
			// Simple tracking - not perfect but sufficient for basic analysis
		}
	}
}

// Helper functions

// stripInlineComment removes trailing inline comments from a line.
func stripInlineComment(line string) string {
	inString := false
	stringChar := byte(0)
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if !inString {
			if ch == '"' || ch == '\'' {
				inString = true
				stringChar = ch
			} else if ch == '/' && i+1 < len(line) && line[i+1] == '/' {
				return strings.TrimSpace(line[:i])
			}
		} else if ch == stringChar {
			inString = false
		}
	}

	return strings.TrimSpace(line)
}

// countBracesOutsideStrings counts opening and closing braces outside string literals.
func countBracesOutsideStrings(line string) (opens int, closes int) {
	inString := false
	stringChar := byte(0)
	escaped := false

	for i := 0; i < len(line); i++ {
		ch := line[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if !inString {
			if ch == '"' || ch == '\'' {
				inString = true
				stringChar = ch
			} else if ch == '{' {
				opens++
			} else if ch == '}' {
				closes++
			}
		} else if ch == stringChar {
			inString = false
		}
	}
	return
}

// truncateStr truncates a string to maxLen characters with ellipsis.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
