package lsp

// InitializeParams for initialize request.
type InitializeParams struct {
	ProcessID             *int               `json:"processId"`
	ClientInfo            *ClientInfo        `json:"clientInfo,omitempty"`
	Locale                string             `json:"locale,omitempty"`
	RootPath              *string            `json:"rootPath,omitempty"`
	RootURI               DocumentURI        `json:"rootUri"`
	InitializationOptions interface{}        `json:"initializationOptions,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities"`
	Trace                 string             `json:"trace,omitempty"`
	WorkspaceFolders      []WorkspaceFolder  `json:"workspaceFolders,omitempty"`
}

// ClientInfo describes the client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// WorkspaceFolder represents a workspace folder.
type WorkspaceFolder struct {
	URI  DocumentURI `json:"uri"`
	Name string      `json:"name"`
}

// ClientCapabilities describes client capabilities.
type ClientCapabilities struct {
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Window       *WindowClientCapabilities       `json:"window,omitempty"`
	General      *GeneralClientCapabilities      `json:"general,omitempty"`
	Experimental interface{}                     `json:"experimental,omitempty"`
}

// WorkspaceClientCapabilities describes workspace capabilities.
type WorkspaceClientCapabilities struct {
	ApplyEdit              bool                               `json:"applyEdit,omitempty"`
	WorkspaceEdit          *WorkspaceEditClientCapabilities   `json:"workspaceEdit,omitempty"`
	DidChangeConfiguration *DidChangeConfigurationCapability  `json:"didChangeConfiguration,omitempty"`
	DidChangeWatchedFiles  *DidChangeWatchedFilesCapability   `json:"didChangeWatchedFiles,omitempty"`
	Symbol                 *WorkspaceSymbolCapability         `json:"symbol,omitempty"`
	ExecuteCommand         *ExecuteCommandCapability          `json:"executeCommand,omitempty"`
	WorkspaceFolders       bool                               `json:"workspaceFolders,omitempty"`
	Configuration          bool                               `json:"configuration,omitempty"`
	SemanticTokens         *SemanticTokensWorkspaceCapability `json:"semanticTokens,omitempty"`
}

// WorkspaceEditClientCapabilities describes workspace edit capabilities.
type WorkspaceEditClientCapabilities struct {
	DocumentChanges       bool     `json:"documentChanges,omitempty"`
	ResourceOperations    []string `json:"resourceOperations,omitempty"`
	FailureHandling       string   `json:"failureHandling,omitempty"`
	NormalizesLineEndings bool     `json:"normalizesLineEndings,omitempty"`
}

// DidChangeConfigurationCapability describes didChangeConfiguration capability.
type DidChangeConfigurationCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DidChangeWatchedFilesCapability describes didChangeWatchedFiles capability.
type DidChangeWatchedFilesCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// WorkspaceSymbolCapability describes workspace symbol capability.
type WorkspaceSymbolCapability struct {
	DynamicRegistration bool                  `json:"dynamicRegistration,omitempty"`
	SymbolKind          *SymbolKindCapability `json:"symbolKind,omitempty"`
	TagSupport          *SymbolTagCapability  `json:"tagSupport,omitempty"`
}

// SymbolKindCapability describes symbol kind capability.
type SymbolKindCapability struct {
	ValueSet []SymbolKind `json:"valueSet,omitempty"`
}

// SymbolTagCapability describes symbol tag capability.
type SymbolTagCapability struct {
	ValueSet []SymbolTag `json:"valueSet,omitempty"`
}

// ExecuteCommandCapability describes execute command capability.
type ExecuteCommandCapability struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// SemanticTokensWorkspaceCapability describes semantic tokens workspace capability.
type SemanticTokensWorkspaceCapability struct {
	RefreshSupport bool `json:"refreshSupport,omitempty"`
}

// TextDocumentClientCapabilities describes text document capabilities.
type TextDocumentClientCapabilities struct {
	Synchronization    *TextDocumentSyncClientCapabilities   `json:"synchronization,omitempty"`
	Completion         *CompletionClientCapabilities         `json:"completion,omitempty"`
	Hover              *HoverClientCapabilities              `json:"hover,omitempty"`
	SignatureHelp      *SignatureHelpClientCapabilities      `json:"signatureHelp,omitempty"`
	Declaration        *DeclarationClientCapabilities        `json:"declaration,omitempty"`
	Definition         *DefinitionClientCapabilities         `json:"definition,omitempty"`
	TypeDefinition     *TypeDefinitionClientCapabilities     `json:"typeDefinition,omitempty"`
	Implementation     *ImplementationClientCapabilities     `json:"implementation,omitempty"`
	References         *ReferenceClientCapabilities          `json:"references,omitempty"`
	DocumentHighlight  *DocumentHighlightClientCapabilities  `json:"documentHighlight,omitempty"`
	DocumentSymbol     *DocumentSymbolClientCapabilities     `json:"documentSymbol,omitempty"`
	CodeAction         *CodeActionClientCapabilities         `json:"codeAction,omitempty"`
	CodeLens           *CodeLensClientCapabilities           `json:"codeLens,omitempty"`
	DocumentLink       *DocumentLinkClientCapabilities       `json:"documentLink,omitempty"`
	ColorProvider      *DocumentColorClientCapabilities      `json:"colorProvider,omitempty"`
	Formatting         *DocumentFormattingClientCapabilities `json:"formatting,omitempty"`
	RangeFormatting    *DocumentRangeFormattingCapabilities  `json:"rangeFormatting,omitempty"`
	OnTypeFormatting   *DocumentOnTypeFormattingCapabilities `json:"onTypeFormatting,omitempty"`
	Rename             *RenameClientCapabilities             `json:"rename,omitempty"`
	PublishDiagnostics *PublishDiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`
	FoldingRange       *FoldingRangeClientCapabilities       `json:"foldingRange,omitempty"`
	SelectionRange     *SelectionRangeClientCapabilities     `json:"selectionRange,omitempty"`
	SemanticTokens     *SemanticTokensClientCapabilities     `json:"semanticTokens,omitempty"`
}

// TextDocumentSyncClientCapabilities describes synchronization capabilities.
type TextDocumentSyncClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	WillSave            bool `json:"willSave,omitempty"`
	WillSaveWaitUntil   bool `json:"willSaveWaitUntil,omitempty"`
	DidSave             bool `json:"didSave,omitempty"`
}

// CompletionClientCapabilities describes completion capabilities.
type CompletionClientCapabilities struct {
	DynamicRegistration bool                          `json:"dynamicRegistration,omitempty"`
	CompletionItem      *CompletionItemCapabilities   `json:"completionItem,omitempty"`
	CompletionItemKind  *CompletionItemKindCapability `json:"completionItemKind,omitempty"`
	ContextSupport      bool                          `json:"contextSupport,omitempty"`
}

// CompletionItemCapabilities describes completion item capabilities.
type CompletionItemCapabilities struct {
	SnippetSupport          bool     `json:"snippetSupport,omitempty"`
	CommitCharactersSupport bool     `json:"commitCharactersSupport,omitempty"`
	DocumentationFormat     []string `json:"documentationFormat,omitempty"`
	DeprecatedSupport       bool     `json:"deprecatedSupport,omitempty"`
	PreselectSupport        bool     `json:"preselectSupport,omitempty"`
	TagSupport              *struct {
		ValueSet []CompletionItemTag `json:"valueSet"`
	} `json:"tagSupport,omitempty"`
	InsertReplaceSupport bool `json:"insertReplaceSupport,omitempty"`
	ResolveSupport       *struct {
		Properties []string `json:"properties"`
	} `json:"resolveSupport,omitempty"`
	InsertTextModeSupport *struct {
		ValueSet []InsertTextMode `json:"valueSet"`
	} `json:"insertTextModeSupport,omitempty"`
}

// CompletionItemKindCapability describes completion item kind capability.
type CompletionItemKindCapability struct {
	ValueSet []CompletionItemKind `json:"valueSet,omitempty"`
}

// HoverClientCapabilities describes hover capabilities.
type HoverClientCapabilities struct {
	DynamicRegistration bool         `json:"dynamicRegistration,omitempty"`
	ContentFormat       []MarkupKind `json:"contentFormat,omitempty"`
}

// SignatureHelpClientCapabilities describes signature help capabilities.
type SignatureHelpClientCapabilities struct {
	DynamicRegistration  bool `json:"dynamicRegistration,omitempty"`
	SignatureInformation *struct {
		DocumentationFormat  []MarkupKind `json:"documentationFormat,omitempty"`
		ParameterInformation *struct {
			LabelOffsetSupport bool `json:"labelOffsetSupport,omitempty"`
		} `json:"parameterInformation,omitempty"`
		ActiveParameterSupport bool `json:"activeParameterSupport,omitempty"`
	} `json:"signatureInformation,omitempty"`
	ContextSupport bool `json:"contextSupport,omitempty"`
}

// DeclarationClientCapabilities describes declaration capabilities.
type DeclarationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// DefinitionClientCapabilities describes definition capabilities.
type DefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// TypeDefinitionClientCapabilities describes type definition capabilities.
type TypeDefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// ImplementationClientCapabilities describes implementation capabilities.
type ImplementationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	LinkSupport         bool `json:"linkSupport,omitempty"`
}

// ReferenceClientCapabilities describes reference capabilities.
type ReferenceClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentHighlightClientCapabilities describes document highlight capabilities.
type DocumentHighlightClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentSymbolClientCapabilities describes document symbol capabilities.
type DocumentSymbolClientCapabilities struct {
	DynamicRegistration               bool                  `json:"dynamicRegistration,omitempty"`
	SymbolKind                        *SymbolKindCapability `json:"symbolKind,omitempty"`
	HierarchicalDocumentSymbolSupport bool                  `json:"hierarchicalDocumentSymbolSupport,omitempty"`
	TagSupport                        *SymbolTagCapability  `json:"tagSupport,omitempty"`
	LabelSupport                      bool                  `json:"labelSupport,omitempty"`
}

// CodeActionClientCapabilities describes code action capabilities.
type CodeActionClientCapabilities struct {
	DynamicRegistration      bool `json:"dynamicRegistration,omitempty"`
	CodeActionLiteralSupport *struct {
		CodeActionKind struct {
			ValueSet []CodeActionKind `json:"valueSet"`
		} `json:"codeActionKind"`
	} `json:"codeActionLiteralSupport,omitempty"`
	IsPreferredSupport bool `json:"isPreferredSupport,omitempty"`
	DisabledSupport    bool `json:"disabledSupport,omitempty"`
	DataSupport        bool `json:"dataSupport,omitempty"`
	ResolveSupport     *struct {
		Properties []string `json:"properties"`
	} `json:"resolveSupport,omitempty"`
	HonorsChangeAnnotations bool `json:"honorsChangeAnnotations,omitempty"`
}

// CodeLensClientCapabilities describes code lens capabilities.
type CodeLensClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentLinkClientCapabilities describes document link capabilities.
type DocumentLinkClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	TooltipSupport      bool `json:"tooltipSupport,omitempty"`
}

// DocumentColorClientCapabilities describes document color capabilities.
type DocumentColorClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentFormattingClientCapabilities describes document formatting capabilities.
type DocumentFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentRangeFormattingCapabilities describes range formatting capabilities.
type DocumentRangeFormattingCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentOnTypeFormattingCapabilities describes on type formatting capabilities.
type DocumentOnTypeFormattingCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// RenameClientCapabilities describes rename capabilities.
type RenameClientCapabilities struct {
	DynamicRegistration           bool `json:"dynamicRegistration,omitempty"`
	PrepareSupport                bool `json:"prepareSupport,omitempty"`
	PrepareSupportDefaultBehavior int  `json:"prepareSupportDefaultBehavior,omitempty"`
	HonorsChangeAnnotations       bool `json:"honorsChangeAnnotations,omitempty"`
}

// PublishDiagnosticsClientCapabilities describes diagnostics capabilities.
type PublishDiagnosticsClientCapabilities struct {
	RelatedInformation bool `json:"relatedInformation,omitempty"`
	TagSupport         *struct {
		ValueSet []DiagnosticTag `json:"valueSet"`
	} `json:"tagSupport,omitempty"`
	VersionSupport         bool `json:"versionSupport,omitempty"`
	CodeDescriptionSupport bool `json:"codeDescriptionSupport,omitempty"`
	DataSupport            bool `json:"dataSupport,omitempty"`
}

// FoldingRangeClientCapabilities describes folding range capabilities.
type FoldingRangeClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	RangeLimit          int  `json:"rangeLimit,omitempty"`
	LineFoldingOnly     bool `json:"lineFoldingOnly,omitempty"`
}

// SelectionRangeClientCapabilities describes selection range capabilities.
type SelectionRangeClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// SemanticTokensClientCapabilities describes semantic tokens capabilities.
type SemanticTokensClientCapabilities struct {
	DynamicRegistration     bool        `json:"dynamicRegistration,omitempty"`
	Requests                interface{} `json:"requests,omitempty"`
	TokenTypes              []string    `json:"tokenTypes,omitempty"`
	TokenModifiers          []string    `json:"tokenModifiers,omitempty"`
	Formats                 []string    `json:"formats,omitempty"`
	OverlappingTokenSupport bool        `json:"overlappingTokenSupport,omitempty"`
	MultilineTokenSupport   bool        `json:"multilineTokenSupport,omitempty"`
}

// WindowClientCapabilities describes window capabilities.
type WindowClientCapabilities struct {
	WorkDoneProgress bool `json:"workDoneProgress,omitempty"`
	ShowMessage      *struct {
		MessageActionItem *struct {
			AdditionalPropertiesSupport bool `json:"additionalPropertiesSupport,omitempty"`
		} `json:"messageActionItem,omitempty"`
	} `json:"showMessage,omitempty"`
	ShowDocument *struct {
		Support bool `json:"support,omitempty"`
	} `json:"showDocument,omitempty"`
}

// GeneralClientCapabilities describes general capabilities.
type GeneralClientCapabilities struct {
	StaleRequestSupport *struct {
		Cancel                 bool     `json:"cancel"`
		RetryOnContentModified []string `json:"retryOnContentModified"`
	} `json:"staleRequestSupport,omitempty"`
	RegularExpressions *struct {
		Engine  string `json:"engine"`
		Version string `json:"version,omitempty"`
	} `json:"regularExpressions,omitempty"`
	Markdown *struct {
		Parser      string   `json:"parser"`
		Version     string   `json:"version,omitempty"`
		AllowedTags []string `json:"allowedTags,omitempty"`
	} `json:"markdown,omitempty"`
}

// InitializeResult is the result of the initialize request.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerInfo describes the server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ServerCapabilities describes server capabilities.
type ServerCapabilities struct {
	TextDocumentSync                 interface{}                      `json:"textDocumentSync,omitempty"`
	CompletionProvider               *CompletionOptions               `json:"completionProvider,omitempty"`
	HoverProvider                    interface{}                      `json:"hoverProvider,omitempty"`
	SignatureHelpProvider            *SignatureHelpOptions            `json:"signatureHelpProvider,omitempty"`
	DeclarationProvider              interface{}                      `json:"declarationProvider,omitempty"`
	DefinitionProvider               interface{}                      `json:"definitionProvider,omitempty"`
	TypeDefinitionProvider           interface{}                      `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider           interface{}                      `json:"implementationProvider,omitempty"`
	ReferencesProvider               interface{}                      `json:"referencesProvider,omitempty"`
	DocumentHighlightProvider        interface{}                      `json:"documentHighlightProvider,omitempty"`
	DocumentSymbolProvider           interface{}                      `json:"documentSymbolProvider,omitempty"`
	CodeActionProvider               interface{}                      `json:"codeActionProvider,omitempty"`
	CodeLensProvider                 *CodeLensOptions                 `json:"codeLensProvider,omitempty"`
	DocumentLinkProvider             *DocumentLinkOptions             `json:"documentLinkProvider,omitempty"`
	ColorProvider                    interface{}                      `json:"colorProvider,omitempty"`
	DocumentFormattingProvider       interface{}                      `json:"documentFormattingProvider,omitempty"`
	DocumentRangeFormattingProvider  interface{}                      `json:"documentRangeFormattingProvider,omitempty"`
	DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider,omitempty"`
	RenameProvider                   interface{}                      `json:"renameProvider,omitempty"`
	FoldingRangeProvider             interface{}                      `json:"foldingRangeProvider,omitempty"`
	ExecuteCommandProvider           *ExecuteCommandOptions           `json:"executeCommandProvider,omitempty"`
	SelectionRangeProvider           interface{}                      `json:"selectionRangeProvider,omitempty"`
	WorkspaceSymbolProvider          interface{}                      `json:"workspaceSymbolProvider,omitempty"`
	Workspace                        *ServerWorkspaceCapabilities     `json:"workspace,omitempty"`
	SemanticTokensProvider           interface{}                      `json:"semanticTokensProvider,omitempty"`
	Experimental                     interface{}                      `json:"experimental,omitempty"`
}

// TextDocumentSyncKind defines how the host syncs with the client.
type TextDocumentSyncKind int

const (
	TextDocumentSyncKindNone        TextDocumentSyncKind = 0
	TextDocumentSyncKindFull        TextDocumentSyncKind = 1
	TextDocumentSyncKindIncremental TextDocumentSyncKind = 2
)

// TextDocumentSyncOptions describes text document sync options.
type TextDocumentSyncOptions struct {
	OpenClose         bool                 `json:"openClose,omitempty"`
	Change            TextDocumentSyncKind `json:"change,omitempty"`
	WillSave          bool                 `json:"willSave,omitempty"`
	WillSaveWaitUntil bool                 `json:"willSaveWaitUntil,omitempty"`
	Save              *SaveOptions         `json:"save,omitempty"`
}

// SaveOptions describes save options.
type SaveOptions struct {
	IncludeText bool `json:"includeText,omitempty"`
}

// CompletionOptions describes completion provider options.
type CompletionOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	AllCommitCharacters []string `json:"allCommitCharacters,omitempty"`
	ResolveProvider     bool     `json:"resolveProvider,omitempty"`
}

// SignatureHelpOptions describes signature help options.
type SignatureHelpOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	RetriggerCharacters []string `json:"retriggerCharacters,omitempty"`
}

// CodeLensOptions describes code lens options.
type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// DocumentLinkOptions describes document link options.
type DocumentLinkOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// DocumentOnTypeFormattingOptions describes on type formatting options.
type DocumentOnTypeFormattingOptions struct {
	FirstTriggerCharacter string   `json:"firstTriggerCharacter"`
	MoreTriggerCharacter  []string `json:"moreTriggerCharacter,omitempty"`
}

// ExecuteCommandOptions describes execute command options.
type ExecuteCommandOptions struct {
	Commands []string `json:"commands,omitempty"`
}

// ServerWorkspaceCapabilities describes server workspace capabilities.
type ServerWorkspaceCapabilities struct {
	WorkspaceFolders *WorkspaceFoldersServerCapabilities `json:"workspaceFolders,omitempty"`
	FileOperations   *FileOperationOptions               `json:"fileOperations,omitempty"`
}

// WorkspaceFoldersServerCapabilities describes workspace folders capabilities.
type WorkspaceFoldersServerCapabilities struct {
	Supported           bool        `json:"supported,omitempty"`
	ChangeNotifications interface{} `json:"changeNotifications,omitempty"`
}

// FileOperationOptions describes file operation options.
type FileOperationOptions struct {
	DidCreate  *FileOperationRegistrationOptions `json:"didCreate,omitempty"`
	WillCreate *FileOperationRegistrationOptions `json:"willCreate,omitempty"`
	DidRename  *FileOperationRegistrationOptions `json:"didRename,omitempty"`
	WillRename *FileOperationRegistrationOptions `json:"willRename,omitempty"`
	DidDelete  *FileOperationRegistrationOptions `json:"didDelete,omitempty"`
	WillDelete *FileOperationRegistrationOptions `json:"willDelete,omitempty"`
}

// FileOperationRegistrationOptions describes file operation registration.
type FileOperationRegistrationOptions struct {
	Filters []FileOperationFilter `json:"filters"`
}

// FileOperationFilter describes a file operation filter.
type FileOperationFilter struct {
	Scheme  string               `json:"scheme,omitempty"`
	Pattern FileOperationPattern `json:"pattern"`
}

// FileOperationPattern describes a file operation pattern.
type FileOperationPattern struct {
	Glob    string `json:"glob"`
	Matches string `json:"matches,omitempty"`
	Options *struct {
		IgnoreCase bool `json:"ignoreCase,omitempty"`
	} `json:"options,omitempty"`
}
