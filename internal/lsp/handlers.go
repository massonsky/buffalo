package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/massonsky/buffalo/pkg/logger"
)

// Text document synchronization handlers

// DidOpenTextDocumentParams for textDocument/didOpen.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams for textDocument/didChange.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// TextDocumentContentChangeEvent describes a change in a text document.
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength int    `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// DidSaveTextDocumentParams for textDocument/didSave.
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         *string                `json:"text,omitempty"`
}

// DidCloseTextDocumentParams for textDocument/didClose.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

func (s *Server) handleTextDocumentDidOpen(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DidOpenTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := &Document{
		URI:        p.TextDocument.URI,
		LanguageID: p.TextDocument.LanguageID,
		Version:    p.TextDocument.Version,
	}
	doc.updateContent(p.TextDocument.Text)
	s.setDocument(doc)

	s.log.Debug("Document opened", logger.String("uri", string(doc.URI)))

	// Analyze and publish diagnostics
	go s.analyzeDocument(doc)

	return nil, nil
}

func (s *Server) handleTextDocumentDidChange(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DidChangeTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	// Apply changes (we use full sync, so just take the last change)
	for _, change := range p.ContentChanges {
		doc.updateContent(change.Text)
	}
	doc.Version = p.TextDocument.Version

	// Analyze and publish diagnostics
	go s.analyzeDocument(doc)

	return nil, nil
}

func (s *Server) handleTextDocumentDidSave(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DidSaveTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	if p.Text != nil {
		doc.updateContent(*p.Text)
	}

	// Full analysis on save
	go s.analyzeDocument(doc)

	return nil, nil
}

func (s *Server) handleTextDocumentDidClose(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DidCloseTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	s.removeDocument(p.TextDocument.URI)
	s.publishDiagnostics(p.TextDocument.URI, []Diagnostic{})

	return nil, nil
}

// analyzeDocument analyzes a document and publishes diagnostics.
func (s *Server) analyzeDocument(doc *Document) {
	diagnostics := s.analyzer.Analyze(doc)
	s.publishDiagnostics(doc.URI, diagnostics)
}

// Completion handlers

func (s *Server) handleCompletion(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p CompletionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	items := s.analyzer.Complete(doc, p.Position, p.Context)

	return CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func (s *Server) handleCompletionResolve(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var item CompletionItem
	if err := json.Unmarshal(params, &item); err != nil {
		return nil, err
	}

	// Add additional documentation if available
	return s.analyzer.ResolveCompletion(item), nil
}

// Hover handler

func (s *Server) handleHover(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	// Check for Buffalo-specific hover info first
	line := doc.getLine(p.Position.Line)

	// Hover on validation rules
	if strings.Contains(line, "buffalo.validate") {
		hover := s.getBuffaloValidationHover(line, p.Position)
		if hover != nil {
			return hover, nil
		}
	}

	// Hover on permission annotations
	if strings.Contains(line, "buffalo.permissions") {
		hover := s.getBuffaloPermissionHover(line, p.Position)
		if hover != nil {
			return hover, nil
		}
	}

	return s.analyzer.Hover(doc, p.Position), nil
}

// getBuffaloValidationHover returns hover info for Buffalo validation rules.
func (s *Server) getBuffaloValidationHover(line string, pos Position) *Hover {
	word := getWordAt(line, pos.Character)

	validationDocs := map[string]string{
		"min_len":   "**min_len** - Minimum string/bytes length\n\n```protobuf\nstring name = 1 [(buffalo.validate.rules).string = {\n  min_len: 1\n}];\n```",
		"max_len":   "**max_len** - Maximum string/bytes length\n\n```protobuf\nstring name = 1 [(buffalo.validate.rules).string = {\n  max_len: 255\n}];\n```",
		"pattern":   "**pattern** - Regex pattern to match\n\n```protobuf\nstring code = 1 [(buffalo.validate.rules).string = {\n  pattern: \"^[A-Z]{2}[0-9]{4}$\"\n}];\n```",
		"email":     "**email** - Must be valid email format\n\n```protobuf\nstring email = 1 [(buffalo.validate.rules).string = {\n  email: true\n}];\n```",
		"uri":       "**uri** - Must be valid URI format\n\n```protobuf\nstring website = 1 [(buffalo.validate.rules).string = {\n  uri: true\n}];\n```",
		"uuid":      "**uuid** - Must be valid UUID format\n\n```protobuf\nstring id = 1 [(buffalo.validate.rules).string = {\n  uuid: true\n}];\n```",
		"gt":        "**gt** - Value must be greater than\n\n```protobuf\nint32 age = 1 [(buffalo.validate.rules).int32 = {\n  gt: 0\n}];\n```",
		"gte":       "**gte** - Value must be greater than or equal to\n\n```protobuf\ndouble lat = 1 [(buffalo.validate.rules).double = {\n  gte: -90.0\n  lte: 90.0\n}];\n```",
		"lt":        "**lt** - Value must be less than\n\n```protobuf\nint32 count = 1 [(buffalo.validate.rules).int32 = {\n  lt: 1000\n}];\n```",
		"lte":       "**lte** - Value must be less than or equal to",
		"in":        "**in** - Value must be one of the specified values\n\n```protobuf\nstring status = 1 [(buffalo.validate.rules).string = {\n  in: [\"active\", \"pending\", \"inactive\"]\n}];\n```",
		"not_in":    "**not_in** - Value must not be one of the specified values",
		"required":  "**required** - Field must be set (non-zero value)\n\n```protobuf\nstring name = 1 [(buffalo.validate.rules) = {\n  required: true\n}];\n```",
		"unique":    "**unique** - All items in repeated field must be unique\n\n```protobuf\nrepeated string tags = 1 [(buffalo.validate.rules).repeated = {\n  unique: true\n}];\n```",
		"min_items": "**min_items** - Minimum number of items in repeated field",
		"max_items": "**max_items** - Maximum number of items in repeated field",
		"gt_now":    "**gt_now** - Timestamp must be in the future",
		"lt_now":    "**lt_now** - Timestamp must be in the past",
	}

	if doc, ok := validationDocs[word]; ok {
		return &Hover{
			Contents: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: doc,
			},
		}
	}

	return nil
}

// getBuffaloPermissionHover returns hover info for Buffalo permission annotations.
func (s *Server) getBuffaloPermissionHover(line string, pos Position) *Hover {
	word := getWordAt(line, pos.Character)

	permissionDocs := map[string]string{
		"action":      "**action** - Permission action name (e.g., \"read\", \"write\", \"delete\")\n\n```protobuf\noption (buffalo.permissions) = {\n  action: \"delete\"\n  roles: [\"admin\"]\n};\n```",
		"roles":       "**roles** - List of roles allowed to access this method\n\n```protobuf\noption (buffalo.permissions) = {\n  roles: [\"admin\", \"moderator\"]\n};\n```",
		"scopes":      "**scopes** - Required OAuth scopes\n\n```protobuf\noption (buffalo.permissions) = {\n  scopes: [\"read:users\", \"write:users\"]\n};\n```",
		"public":      "**public** - Make endpoint public (no authentication required)\n\n```protobuf\nrpc HealthCheck(Empty) returns (HealthResponse) {\n  option (buffalo.permissions) = {\n    public: true\n  };\n}\n```",
		"allow_self":  "**allow_self** - Allow users to access their own resources\n\n```protobuf\nrpc GetProfile(GetProfileRequest) returns (User) {\n  option (buffalo.permissions) = {\n    action: \"read\"\n    allow_self: true\n  };\n}\n```",
		"require_mfa": "**require_mfa** - Require multi-factor authentication\n\n```protobuf\nrpc TransferFunds(TransferRequest) returns (TransferResponse) {\n  option (buffalo.permissions) = {\n    require_mfa: true\n  };\n}\n```",
		"audit_log":   "**audit_log** - Enable audit logging for this method\n\n```protobuf\noption (buffalo.permissions) = {\n  audit_log: true\n};\n```",
		"rate_limit":  "**rate_limit** - Configure rate limiting\n\n```protobuf\noption (buffalo.permissions) = {\n  rate_limit: {\n    requests: 100\n    window: \"1m\"\n    per_user: true\n  }\n};\n```",
		"resource":    "**resource** - Service resource name (used in permission matrix)\n\n```protobuf\nservice UserService {\n  option (buffalo.permissions.resource) = \"users\";\n}\n```",
	}

	if doc, ok := permissionDocs[word]; ok {
		return &Hover{
			Contents: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: doc,
			},
		}
	}

	return nil
}

// getWordAt extracts the word at the given character position.
func getWordAt(line string, char int) string {
	if char < 0 || char >= len(line) {
		return ""
	}

	start := char
	for start > 0 && isWordCharByte(line[start-1]) {
		start--
	}

	end := char
	for end < len(line) && isWordCharByte(line[end]) {
		end++
	}

	return line[start:end]
}

func isWordCharByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// Definition handler

func (s *Server) handleDefinition(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p TextDocumentPositionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	// Check if cursor is on an import path
	line := doc.getLine(p.Position.Line)
	if strings.Contains(line, "import") && strings.Contains(line, "\"") {
		// Extract import path
		start := strings.Index(line, "\"")
		end := strings.LastIndex(line, "\"")
		if start >= 0 && end > start && p.Position.Character > start && p.Position.Character <= end {
			importPath := line[start+1 : end]

			// Check if it's an embedded Buffalo proto
			if content, exists := s.analyzer.GetEmbeddedProto(importPath); exists {
				// Create a virtual document for the embedded proto
				virtualURI := DocumentURI("buffalo-embedded:///" + importPath)

				// Check if we already have this document
				embeddedDoc := s.getDocument(virtualURI)
				if embeddedDoc == nil {
					embeddedDoc = NewDocument(virtualURI, content)
					s.setDocument(embeddedDoc)
				}

				return &Location{
					URI: virtualURI,
					Range: Range{
						Start: Position{Line: 0, Character: 0},
						End:   Position{Line: 0, Character: 0},
					},
				}, nil
			}
		}
	}

	return s.analyzer.Definition(doc, p.Position), nil
}

// References handler

// ReferenceParams for textDocument/references.
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// ReferenceContext contains additional reference information.
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

func (s *Server) handleReferences(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p ReferenceParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	// Get all documents for cross-file references
	s.documentsMu.RLock()
	docs := make([]*Document, 0, len(s.documents))
	for _, d := range s.documents {
		docs = append(docs, d)
	}
	s.documentsMu.RUnlock()

	return s.analyzer.References(doc, p.Position, docs, p.Context.IncludeDeclaration), nil
}

// Document symbol handler

// DocumentSymbolParams for textDocument/documentSymbol.
type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

func (s *Server) handleDocumentSymbol(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DocumentSymbolParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return s.analyzer.DocumentSymbols(doc), nil
}

// Formatting handler

func (s *Server) handleFormatting(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DocumentFormattingParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return s.analyzer.Format(doc, p.Options), nil
}

// Code action handler

func (s *Server) handleCodeAction(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p CodeActionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return s.analyzer.CodeActions(doc, p.Range, p.Context), nil
}

// Rename handlers

// PrepareRenameParams for textDocument/prepareRename.
type PrepareRenameParams struct {
	TextDocumentPositionParams
}

// PrepareRenameResult is the result of prepareRename.
type PrepareRenameResult struct {
	Range       Range  `json:"range"`
	Placeholder string `json:"placeholder"`
}

func (s *Server) handlePrepareRename(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p PrepareRenameParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return s.analyzer.PrepareRename(doc, p.Position), nil
}

func (s *Server) handleRename(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p RenameParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	// Get all documents for cross-file rename
	s.documentsMu.RLock()
	docs := make([]*Document, 0, len(s.documents))
	for _, d := range s.documents {
		docs = append(docs, d)
	}
	s.documentsMu.RUnlock()

	return s.analyzer.Rename(doc, p.Position, p.NewName, docs), nil
}

// Folding range handler

// FoldingRangeParams for textDocument/foldingRange.
type FoldingRangeParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

func (s *Server) handleFoldingRange(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p FoldingRangeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return s.analyzer.FoldingRanges(doc), nil
}

// Semantic tokens handler

// SemanticTokensParams for textDocument/semanticTokens/full.
type SemanticTokensParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

func (s *Server) handleSemanticTokensFull(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p SemanticTokensParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	doc := s.getDocument(p.TextDocument.URI)
	if doc == nil {
		return nil, nil
	}

	return s.analyzer.SemanticTokens(doc), nil
}

// Workspace handlers

// DidChangeConfigurationParams for workspace/didChangeConfiguration.
type DidChangeConfigurationParams struct {
	Settings interface{} `json:"settings"`
}

func (s *Server) handleDidChangeConfiguration(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DidChangeConfigurationParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	// TODO: Apply configuration changes

	return nil, nil
}

// DidChangeWatchedFilesParams for workspace/didChangeWatchedFiles.
type DidChangeWatchedFilesParams struct {
	Changes []FileEvent `json:"changes"`
}

// FileEvent describes a file change event.
type FileEvent struct {
	URI  DocumentURI    `json:"uri"`
	Type FileChangeType `json:"type"`
}

// FileChangeType describes the type of a file change.
type FileChangeType int

const (
	FileChangeTypeCreated FileChangeType = 1
	FileChangeTypeChanged FileChangeType = 2
	FileChangeTypeDeleted FileChangeType = 3
)

func (s *Server) handleDidChangeWatchedFiles(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DidChangeWatchedFilesParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	for _, change := range p.Changes {
		if strings.HasSuffix(string(change.URI), ".proto") {
			// Re-analyze proto files
			if doc := s.getDocument(change.URI); doc != nil {
				go s.analyzeDocument(doc)
			}
		}
	}

	return nil, nil
}

// Helper to avoid import cycle
func logger_String(key, value string) interface{} {
	return struct {
		Key   string
		Value string
	}{key, value}
}
