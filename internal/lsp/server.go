package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/massonsky/buffalo/internal/version"
	"github.com/massonsky/buffalo/pkg/logger"
)

// Server is the LSP server for Buffalo.
type Server struct {
	log         *logger.Logger
	reader      *bufio.Reader
	writer      io.Writer
	mu          sync.Mutex
	initialized bool
	shutdown    bool

	// Document state
	documents   map[DocumentURI]*Document
	documentsMu sync.RWMutex

	// Workspace state
	workspaceFolders []WorkspaceFolder
	rootURI          DocumentURI

	// Handlers
	handlers map[string]Handler

	// Proto analysis
	analyzer *ProtoAnalyzer

	// Server capabilities
	capabilities ServerCapabilities
}

// Handler is a function that handles an LSP request.
type Handler func(ctx context.Context, params json.RawMessage) (interface{}, error)

// Document represents an open document.
type Document struct {
	URI        DocumentURI
	LanguageID string
	Version    int
	Content    string
	Lines      []string
}

// NewServer creates a new LSP server.
func NewServer(log *logger.Logger) *Server {
	s := &Server{
		log:       log,
		documents: make(map[DocumentURI]*Document),
		handlers:  make(map[string]Handler),
	}

	s.analyzer = NewProtoAnalyzer(log)
	s.registerHandlers()
	s.initCapabilities()

	return s
}

// initCapabilities initializes server capabilities.
func (s *Server) initCapabilities() {
	s.capabilities = ServerCapabilities{
		TextDocumentSync: &TextDocumentSyncOptions{
			OpenClose: true,
			Change:    TextDocumentSyncKindFull,
			Save: &SaveOptions{
				IncludeText: true,
			},
		},
		CompletionProvider: &CompletionOptions{
			TriggerCharacters: []string{".", "(", "[", "=", " ", "\"", "/"},
			ResolveProvider:   true,
		},
		HoverProvider:              true,
		DefinitionProvider:         true,
		ReferencesProvider:         true,
		DocumentSymbolProvider:     true,
		DocumentFormattingProvider: true,
		CodeActionProvider: &CodeActionOptions{
			CodeActionKinds: []CodeActionKind{
				CodeActionKindQuickFix,
				CodeActionKindRefactor,
				CodeActionKindSource,
			},
		},
		RenameProvider: &RenameOptions{
			PrepareProvider: true,
		},
		FoldingRangeProvider: true,
		SemanticTokensProvider: &SemanticTokensOptions{
			Legend: SemanticTokensLegend{
				TokenTypes: []string{
					"namespace", "type", "class", "enum", "interface",
					"struct", "typeParameter", "parameter", "variable",
					"property", "enumMember", "event", "function", "method",
					"macro", "keyword", "modifier", "comment", "string",
					"number", "regexp", "operator",
				},
				TokenModifiers: []string{
					"declaration", "definition", "readonly", "static",
					"deprecated", "abstract", "async", "modification",
					"documentation", "defaultLibrary",
				},
			},
			Full:  true,
			Range: false,
		},
	}
}

// CodeActionOptions describes code action options.
type CodeActionOptions struct {
	CodeActionKinds []CodeActionKind `json:"codeActionKinds,omitempty"`
	ResolveProvider bool             `json:"resolveProvider,omitempty"`
}

// RenameOptions describes rename options.
type RenameOptions struct {
	PrepareProvider bool `json:"prepareProvider,omitempty"`
}

// SemanticTokensOptions describes semantic tokens options.
type SemanticTokensOptions struct {
	Legend SemanticTokensLegend `json:"legend"`
	Range  bool                 `json:"range,omitempty"`
	Full   interface{}          `json:"full,omitempty"`
}

// registerHandlers registers all request handlers.
func (s *Server) registerHandlers() {
	// Lifecycle
	s.handlers["initialize"] = s.handleInitialize
	s.handlers["initialized"] = s.handleInitialized
	s.handlers["shutdown"] = s.handleShutdown
	s.handlers["exit"] = s.handleExit

	// Text document synchronization
	s.handlers["textDocument/didOpen"] = s.handleTextDocumentDidOpen
	s.handlers["textDocument/didChange"] = s.handleTextDocumentDidChange
	s.handlers["textDocument/didSave"] = s.handleTextDocumentDidSave
	s.handlers["textDocument/didClose"] = s.handleTextDocumentDidClose

	// Language features
	s.handlers["textDocument/completion"] = s.handleCompletion
	s.handlers["completionItem/resolve"] = s.handleCompletionResolve
	s.handlers["textDocument/hover"] = s.handleHover
	s.handlers["textDocument/definition"] = s.handleDefinition
	s.handlers["textDocument/references"] = s.handleReferences
	s.handlers["textDocument/documentSymbol"] = s.handleDocumentSymbol
	s.handlers["textDocument/formatting"] = s.handleFormatting
	s.handlers["textDocument/codeAction"] = s.handleCodeAction
	s.handlers["textDocument/rename"] = s.handleRename
	s.handlers["textDocument/prepareRename"] = s.handlePrepareRename
	s.handlers["textDocument/foldingRange"] = s.handleFoldingRange
	s.handlers["textDocument/semanticTokens/full"] = s.handleSemanticTokensFull

	// Workspace
	s.handlers["workspace/didChangeConfiguration"] = s.handleDidChangeConfiguration
	s.handlers["workspace/didChangeWatchedFiles"] = s.handleDidChangeWatchedFiles
}

// ServeStdio starts the server on stdin/stdout.
func (s *Server) ServeStdio(ctx context.Context) error {
	s.log.Info("Starting Buffalo LSP server on stdio")
	s.reader = bufio.NewReader(os.Stdin)
	s.writer = os.Stdout
	return s.serve(ctx)
}

// ServeTCP starts the server on TCP.
func (s *Server) ServeTCP(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	defer listener.Close()

	s.log.Info("Starting Buffalo LSP server on TCP", logger.String("address", addr))

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			s.log.Error("Failed to accept connection", logger.Any("error", err))
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			s.reader = bufio.NewReader(c)
			s.writer = c
			if err := s.serve(ctx); err != nil {
				s.log.Error("Connection error", logger.Any("error", err))
			}
		}(conn)
	}
}

// serve is the main message loop.
func (s *Server) serve(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			s.log.Error("Failed to read message", logger.Any("error", err))
			continue
		}

		go s.handleMessage(ctx, msg)
	}
}

// readMessage reads a single LSP message.
func (s *Server) readMessage() (json.RawMessage, error) {
	// Read headers
	var contentLength int
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)

		if line == "" {
			break
		}

		if strings.HasPrefix(line, "Content-Length:") {
			length := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, err = strconv.Atoi(length)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %w", err)
			}
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	// Read content
	content := make([]byte, contentLength)
	_, err := io.ReadFull(s.reader, content)
	if err != nil {
		return nil, err
	}

	return content, nil
}

// writeMessage writes an LSP message.
func (s *Server) writeMessage(msg interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	content, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(content))
	if _, err := s.writer.Write([]byte(header)); err != nil {
		return err
	}
	_, err = s.writer.Write(content)
	return err
}

// handleMessage handles a single message.
func (s *Server) handleMessage(ctx context.Context, content json.RawMessage) {
	// Try to parse as request
	var req RequestMessage
	if err := json.Unmarshal(content, &req); err != nil {
		s.log.Error("Failed to parse message", logger.Any("error", err))
		return
	}

	// Check if it's a notification (no ID)
	if req.ID == nil {
		s.handleNotification(ctx, req.Method, req.Params)
		return
	}

	// Handle request
	s.handleRequest(ctx, req)
}

// handleRequest handles a request message.
func (s *Server) handleRequest(ctx context.Context, req RequestMessage) {
	s.log.Debug("Handling request", logger.String("method", req.Method))

	handler, ok := s.handlers[req.Method]
	if !ok {
		s.sendError(req.ID, MethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
		return
	}

	result, err := handler(ctx, req.Params)
	if err != nil {
		s.sendError(req.ID, InternalError, err.Error())
		return
	}

	s.sendResult(req.ID, result)
}

// handleNotification handles a notification message.
func (s *Server) handleNotification(ctx context.Context, method string, params json.RawMessage) {
	s.log.Debug("Handling notification", logger.String("method", method))

	handler, ok := s.handlers[method]
	if !ok {
		s.log.Warn("Unknown notification", logger.String("method", method))
		return
	}

	if _, err := handler(ctx, params); err != nil {
		s.log.Error("Notification handler error",
			logger.String("method", method),
			logger.Any("error", err))
	}
}

// sendResult sends a success response.
func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := ResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	if err := s.writeMessage(resp); err != nil {
		s.log.Error("Failed to send result", logger.Any("error", err))
	}
}

// sendError sends an error response.
func (s *Server) sendError(id interface{}, code int, message string) {
	resp := ResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &ResponseError{
			Code:    code,
			Message: message,
		},
	}
	if err := s.writeMessage(resp); err != nil {
		s.log.Error("Failed to send error", logger.Any("error", err))
	}
}

// sendNotification sends a notification.
func (s *Server) sendNotification(method string, params interface{}) {
	paramsJSON, _ := json.Marshal(params)
	notif := NotificationMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}
	if err := s.writeMessage(notif); err != nil {
		s.log.Error("Failed to send notification", logger.Any("error", err))
	}
}

// publishDiagnostics publishes diagnostics for a document.
func (s *Server) publishDiagnostics(uri DocumentURI, diagnostics []Diagnostic) {
	s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

// Lifecycle handlers

func (s *Server) handleInitialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p InitializeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}

	s.rootURI = p.RootURI
	s.workspaceFolders = p.WorkspaceFolders

	s.log.Info("Initializing Buffalo LSP",
		logger.String("rootUri", string(p.RootURI)),
		logger.Int("workspaceFolders", len(p.WorkspaceFolders)))

	return InitializeResult{
		Capabilities: s.capabilities,
		ServerInfo: &ServerInfo{
			Name:    "buffalo-lsp",
			Version: version.Version,
		},
	}, nil
}

func (s *Server) handleInitialized(ctx context.Context, params json.RawMessage) (interface{}, error) {
	s.initialized = true
	s.log.Info("Buffalo LSP initialized")
	return nil, nil
}

func (s *Server) handleShutdown(ctx context.Context, params json.RawMessage) (interface{}, error) {
	s.shutdown = true
	s.log.Info("Shutting down Buffalo LSP")
	return nil, nil
}

func (s *Server) handleExit(ctx context.Context, params json.RawMessage) (interface{}, error) {
	s.log.Info("Exiting Buffalo LSP")
	if s.shutdown {
		os.Exit(0)
	}
	os.Exit(1)
	return nil, nil
}

// getDocument returns a document by URI.
func (s *Server) getDocument(uri DocumentURI) *Document {
	s.documentsMu.RLock()
	defer s.documentsMu.RUnlock()
	return s.documents[uri]
}

// setDocument stores a document.
func (s *Server) setDocument(doc *Document) {
	s.documentsMu.Lock()
	defer s.documentsMu.Unlock()
	s.documents[doc.URI] = doc
}

// removeDocument removes a document.
func (s *Server) removeDocument(uri DocumentURI) {
	s.documentsMu.Lock()
	defer s.documentsMu.Unlock()
	delete(s.documents, uri)
}

// updateDocumentContent updates document content and recomputes lines.
func (doc *Document) updateContent(content string) {
	doc.Content = content
	doc.Lines = strings.Split(content, "\n")
}

// NewDocument creates a new document with the given URI and content.
func NewDocument(uri DocumentURI, content string) *Document {
	doc := &Document{
		URI:     uri,
		Version: 1,
	}
	doc.updateContent(content)
	return doc
}

// getLine returns a line by index (0-based).
func (doc *Document) getLine(line int) string {
	if line < 0 || line >= len(doc.Lines) {
		return ""
	}
	return doc.Lines[line]
}

// getWordAtPosition returns the word at the given position.
func (doc *Document) getWordAtPosition(pos Position) string {
	line := doc.getLine(pos.Line)
	if line == "" {
		return ""
	}

	// Find word boundaries
	start := pos.Character
	end := pos.Character

	for start > 0 && isWordChar(rune(line[start-1])) {
		start--
	}
	for end < len(line) && isWordChar(rune(line[end])) {
		end++
	}

	if start >= end {
		return ""
	}
	return line[start:end]
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_' || r == '.'
}
