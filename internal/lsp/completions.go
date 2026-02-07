package lsp

import "strings"

// topLevelCompletions returns completions for top-level proto constructs.
func topLevelCompletions() []CompletionItem {
	return []CompletionItem{
		{
			Label:            "syntax",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Syntax declaration",
			InsertText:       "syntax = \"proto3\";",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Specifies the protobuf syntax version.\n\n```protobuf\nsyntax = \"proto3\";\n```",
			},
		},
		{
			Label:            "package",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Package declaration",
			InsertText:       "package ${1:package_name};",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Declares the package namespace for the proto file.\n\n```protobuf\npackage mypackage;\n```",
			},
		},
		{
			Label:            "import",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Import statement",
			InsertText:       "import \"${1:path/to/file.proto}\";",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Imports definitions from another proto file.\n\n```protobuf\nimport \"google/protobuf/empty.proto\";\n```",
			},
		},
		{
			Label:            "import public",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Public import",
			InsertText:       "import public \"${1:path/to/file.proto}\";",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Public import - transitive import for dependents.",
			},
		},
		{
			Label:            "option",
			Kind:             CompletionItemKindKeyword,
			Detail:           "File option",
			InsertText:       "option ${1:option_name} = ${2:value};",
			InsertTextFormat: InsertTextFormatSnippet,
		},
		{
			Label:            "message",
			Kind:             CompletionItemKindClass,
			Detail:           "Message definition",
			InsertText:       "message ${1:MessageName} {\n  $0\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Define a new message type.\n\n```protobuf\nmessage User {\n  string name = 1;\n  int32 age = 2;\n}\n```",
			},
		},
		{
			Label:            "service",
			Kind:             CompletionItemKindInterface,
			Detail:           "gRPC service definition",
			InsertText:       "service ${1:ServiceName} {\n  rpc ${2:MethodName}(${3:Request}) returns (${4:Response});\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Define a gRPC service with RPC methods.\n\n```protobuf\nservice UserService {\n  rpc GetUser(GetUserRequest) returns (User);\n  rpc CreateUser(CreateUserRequest) returns (User);\n}\n```",
			},
		},
		{
			Label:            "enum",
			Kind:             CompletionItemKindEnum,
			Detail:           "Enum definition",
			InsertText:       "enum ${1:EnumName} {\n  ${2:UNKNOWN} = 0;\n  $0\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Define an enumeration type.\n\n```protobuf\nenum Status {\n  STATUS_UNKNOWN = 0;\n  STATUS_ACTIVE = 1;\n  STATUS_INACTIVE = 2;\n}\n```",
			},
		},
		{
			Label:            "extend",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Extend a message",
			InsertText:       "extend ${1:MessageName} {\n  $0\n}",
			InsertTextFormat: InsertTextFormatSnippet,
		},
	}
}

// fieldTypeCompletions returns completions for field types.
func fieldTypeCompletions() []CompletionItem {
	return []CompletionItem{
		// Scalar types
		{Label: "double", Kind: CompletionItemKindKeyword, Detail: "64-bit floating point", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "64-bit IEEE 754 floating point number."}},
		{Label: "float", Kind: CompletionItemKindKeyword, Detail: "32-bit floating point", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "32-bit IEEE 754 floating point number."}},
		{Label: "int32", Kind: CompletionItemKindKeyword, Detail: "32-bit signed integer", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Uses variable-length encoding. Inefficient for negative numbers – use sint32 if negatives are likely."}},
		{Label: "int64", Kind: CompletionItemKindKeyword, Detail: "64-bit signed integer", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Uses variable-length encoding. Inefficient for negative numbers – use sint64 if negatives are likely."}},
		{Label: "uint32", Kind: CompletionItemKindKeyword, Detail: "32-bit unsigned integer", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Uses variable-length encoding."}},
		{Label: "uint64", Kind: CompletionItemKindKeyword, Detail: "64-bit unsigned integer", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Uses variable-length encoding."}},
		{Label: "sint32", Kind: CompletionItemKindKeyword, Detail: "32-bit signed (ZigZag)", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Uses ZigZag encoding. More efficient than int32 for negative values."}},
		{Label: "sint64", Kind: CompletionItemKindKeyword, Detail: "64-bit signed (ZigZag)", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Uses ZigZag encoding. More efficient than int64 for negative values."}},
		{Label: "fixed32", Kind: CompletionItemKindKeyword, Detail: "32-bit fixed", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Always 4 bytes. More efficient than uint32 if values are often greater than 2^28."}},
		{Label: "fixed64", Kind: CompletionItemKindKeyword, Detail: "64-bit fixed", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Always 8 bytes. More efficient than uint64 if values are often greater than 2^56."}},
		{Label: "sfixed32", Kind: CompletionItemKindKeyword, Detail: "32-bit signed fixed", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Always 4 bytes."}},
		{Label: "sfixed64", Kind: CompletionItemKindKeyword, Detail: "64-bit signed fixed", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Always 8 bytes."}},
		{Label: "bool", Kind: CompletionItemKindKeyword, Detail: "Boolean", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "Boolean value (true/false)."}},
		{Label: "string", Kind: CompletionItemKindKeyword, Detail: "UTF-8 string", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "UTF-8 encoded or 7-bit ASCII text. Maximum 2GB."}},
		{Label: "bytes", Kind: CompletionItemKindKeyword, Detail: "Arbitrary bytes", Documentation: MarkupContent{Kind: MarkupKindMarkdown, Value: "May contain any arbitrary sequence of bytes. Maximum 2GB."}},
		// Complex types
		{
			Label:            "map",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Map field",
			InsertText:       "map<${1:key_type}, ${2:value_type}> ${3:field_name} = ${4:number};",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Map field with key-value pairs.\n\n```protobuf\nmap<string, int32> scores = 1;\n```",
			},
		},
		{
			Label:            "oneof",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Oneof field group",
			InsertText:       "oneof ${1:name} {\n  ${2:type} ${3:field} = ${4:number};\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Group of fields where only one can be set at a time.\n\n```protobuf\noneof result {\n  string error = 1;\n  Data data = 2;\n}\n```",
			},
		},
		// Well-known types
		{
			Label:      "google.protobuf.Timestamp",
			Kind:       CompletionItemKindClass,
			Detail:     "Timestamp (WKT)",
			InsertText: "google.protobuf.Timestamp",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "A point in time with nanosecond precision.\n\nRequires: `import \"google/protobuf/timestamp.proto\";`",
			},
		},
		{
			Label:      "google.protobuf.Duration",
			Kind:       CompletionItemKindClass,
			Detail:     "Duration (WKT)",
			InsertText: "google.protobuf.Duration",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "A signed, fixed-length span of time.\n\nRequires: `import \"google/protobuf/duration.proto\";`",
			},
		},
		{
			Label:      "google.protobuf.Any",
			Kind:       CompletionItemKindClass,
			Detail:     "Any (WKT)",
			InsertText: "google.protobuf.Any",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Container for an arbitrary message type.\n\nRequires: `import \"google/protobuf/any.proto\";`",
			},
		},
		{
			Label:      "google.protobuf.Struct",
			Kind:       CompletionItemKindClass,
			Detail:     "Struct (WKT)",
			InsertText: "google.protobuf.Struct",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Dynamic JSON-like structure.\n\nRequires: `import \"google/protobuf/struct.proto\";`",
			},
		},
	}
}

// fieldModifierCompletions returns completions for field modifiers.
func fieldModifierCompletions() []CompletionItem {
	return []CompletionItem{
		{
			Label:            "repeated",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Repeated field (array)",
			InsertText:       "repeated ${1:type} ${2:name} = ${3:number};",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "A repeated field (list/array) of values.\n\n```protobuf\nrepeated string tags = 1;\n```",
			},
		},
		{
			Label:            "optional",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Optional field (proto2/proto3)",
			InsertText:       "optional ${1:type} ${2:name} = ${3:number};",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "An optional field. In proto3, enables presence tracking.",
			},
		},
		{
			Label:            "reserved",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Reserved field numbers/names",
			InsertText:       "reserved ${1:2, 15, 9 to 11};",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Reserve field numbers or names for removed fields.\n\n```protobuf\nreserved 2, 15, 9 to 11;\nreserved \"foo\", \"bar\";\n```",
			},
		},
	}
}

// serviceCompletions returns completions for inside a service.
func serviceCompletions() []CompletionItem {
	return []CompletionItem{
		{
			Label:            "rpc",
			Kind:             CompletionItemKindMethod,
			Detail:           "RPC method",
			InsertText:       "rpc ${1:MethodName}(${2:Request}) returns (${3:Response});",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Define an RPC method.\n\n```protobuf\nrpc GetUser(GetUserRequest) returns (User);\n```",
			},
		},
		{
			Label:            "rpc stream",
			Kind:             CompletionItemKindMethod,
			Detail:           "Streaming RPC",
			InsertText:       "rpc ${1:MethodName}(stream ${2:Request}) returns (stream ${3:Response});",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Define a bidirectional streaming RPC.\n\n```protobuf\nrpc Chat(stream Message) returns (stream Message);\n```",
			},
		},
		{
			Label:            "rpc server-stream",
			Kind:             CompletionItemKindMethod,
			Detail:           "Server streaming RPC",
			InsertText:       "rpc ${1:MethodName}(${2:Request}) returns (stream ${3:Response});",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Server-side streaming RPC.\n\n```protobuf\nrpc ListUsers(ListRequest) returns (stream User);\n```",
			},
		},
		{
			Label:            "rpc client-stream",
			Kind:             CompletionItemKindMethod,
			Detail:           "Client streaming RPC",
			InsertText:       "rpc ${1:MethodName}(stream ${2:Request}) returns (${3:Response});",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Client-side streaming RPC.\n\n```protobuf\nrpc UploadFiles(stream File) returns (UploadResult);\n```",
			},
		},
		{
			Label:            "option",
			Kind:             CompletionItemKindKeyword,
			Detail:           "Service/method option",
			InsertText:       "option (${1:option_name}) = ${2:value};",
			InsertTextFormat: InsertTextFormatSnippet,
		},
	}
}

// buffaloAnnotationCompletions returns completions for Buffalo annotations.
func buffaloAnnotationCompletions() []CompletionItem {
	return []CompletionItem{
		// Validation annotations
		{
			Label:            "buffalo.validate.rules",
			Kind:             CompletionItemKindProperty,
			Detail:           "Field validation rules",
			InsertText:       "(buffalo.validate.rules).${1|string,int,float,bool,bytes,repeated,map|} = {$0}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Add validation rules to a field.\n\n```protobuf\nstring email = 1 [(buffalo.validate.rules).string = {\n  email: true\n  max_len: 255\n}];\n```",
			},
		},
		{
			Label:            "(buffalo.validate.rules).string",
			Kind:             CompletionItemKindProperty,
			Detail:           "String validation",
			InsertText:       "(buffalo.validate.rules).string = {\n  ${1|min_len,max_len,len,pattern,email,uri,uuid,ip,hostname|}: ${2:value}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
		},
		{
			Label:            "(buffalo.validate.rules).int",
			Kind:             CompletionItemKindProperty,
			Detail:           "Integer validation",
			InsertText:       "(buffalo.validate.rules).int = {\n  ${1|gt,gte,lt,lte,in,not_in|}: ${2:value}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
		},
		{
			Label:            "(buffalo.validate.rules).message",
			Kind:             CompletionItemKindProperty,
			Detail:           "Message validation",
			InsertText:       "(buffalo.validate.rules).message = {\n  required: ${1:true}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
		},
		{
			Label:            "(buffalo.validate.rules).repeated",
			Kind:             CompletionItemKindProperty,
			Detail:           "Repeated field validation",
			InsertText:       "(buffalo.validate.rules).repeated = {\n  min_items: ${1:1}\n  max_items: ${2:100}\n  unique: ${3:true}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
		},
		// Permission annotations
		{
			Label:            "buffalo.permissions.required",
			Kind:             CompletionItemKindProperty,
			Detail:           "Required permissions",
			InsertText:       "(buffalo.permissions.required) = \"${1:permission.name}\"",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Specify required permissions for an RPC method.\n\n```protobuf\nrpc DeleteUser(DeleteUserRequest) returns (Empty) {\n  option (buffalo.permissions.required) = \"users.delete\";\n}\n```",
			},
		},
		{
			Label:            "buffalo.permissions.any",
			Kind:             CompletionItemKindProperty,
			Detail:           "Any of permissions",
			InsertText:       "(buffalo.permissions.any) = [\"${1:perm1}\", \"${2:perm2}\"]",
			InsertTextFormat: InsertTextFormatSnippet,
		},
		{
			Label:            "buffalo.permissions.all",
			Kind:             CompletionItemKindProperty,
			Detail:           "All permissions required",
			InsertText:       "(buffalo.permissions.all) = [\"${1:perm1}\", \"${2:perm2}\"]",
			InsertTextFormat: InsertTextFormatSnippet,
		},
		// Extended permission annotations
		{
			Label:            "(buffalo.permissions)",
			Kind:             CompletionItemKindProperty,
			Detail:           "Full permission block",
			InsertText:       "(buffalo.permissions) = {\n  action: \"${1:read}\"\n  roles: [\"${2:admin}\"]\n  ${3:audit_log: true}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Define full permission configuration for an RPC method.\n\n```protobuf\nrpc GetUser(GetUserRequest) returns (User) {\n  option (buffalo.permissions) = {\n    action: \"read\"\n    roles: [\"admin\", \"user\"]\n    audit_log: true\n  };\n}\n```",
			},
		},
		{
			Label:            "buffalo.permissions.public",
			Kind:             CompletionItemKindProperty,
			Detail:           "Public endpoint (no auth)",
			InsertText:       "(buffalo.permissions) = {\n  public: true\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Mark an RPC method as public (no authentication required).\n\n```protobuf\nrpc HealthCheck(Empty) returns (HealthResponse) {\n  option (buffalo.permissions) = { public: true };\n}\n```",
			},
		},
		{
			Label:            "buffalo.permissions.resource",
			Kind:             CompletionItemKindProperty,
			Detail:           "Service resource name",
			InsertText:       "(buffalo.permissions.resource) = \"${1:users}\"",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Define the resource name for a service (used in permission matrix).\n\n```protobuf\nservice UserService {\n  option (buffalo.permissions.resource) = \"users\";\n}\n```",
			},
		},
		{
			Label:            "buffalo.permissions.allow_self",
			Kind:             CompletionItemKindProperty,
			Detail:           "Allow self-access",
			InsertText:       "allow_self: true",
			InsertTextFormat: InsertTextFormatPlainText,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Allow users to access their own resources.\n\n```protobuf\noption (buffalo.permissions) = {\n  action: \"read\"\n  allow_self: true\n};\n```",
			},
		},
		{
			Label:            "buffalo.permissions.require_mfa",
			Kind:             CompletionItemKindProperty,
			Detail:           "Require MFA",
			InsertText:       "require_mfa: true",
			InsertTextFormat: InsertTextFormatPlainText,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Require multi-factor authentication for this method.",
			},
		},
		{
			Label:            "buffalo.permissions.audit_log",
			Kind:             CompletionItemKindProperty,
			Detail:           "Enable audit logging",
			InsertText:       "audit_log: true",
			InsertTextFormat: InsertTextFormatPlainText,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Enable audit logging for this method.",
			},
		},
		{
			Label:            "buffalo.permissions.scopes",
			Kind:             CompletionItemKindProperty,
			Detail:           "OAuth scopes",
			InsertText:       "scopes: [\"${1:read:users}\"]",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Required OAuth scopes for this method.\n\n```protobuf\noption (buffalo.permissions) = {\n  scopes: [\"read:users\", \"write:users\"]\n};\n```",
			},
		},
		{
			Label:            "buffalo.permissions.rate_limit",
			Kind:             CompletionItemKindProperty,
			Detail:           "Rate limiting",
			InsertText:       "rate_limit: {\n  requests: ${1:100}\n  window: \"${2:1m}\"\n  per_user: ${3:true}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Configure rate limiting for this method.\n\n```protobuf\noption (buffalo.permissions) = {\n  rate_limit: {\n    requests: 100\n    window: \"1m\"\n    per_user: true\n  }\n};\n```",
			},
		},
		// Extended validation annotations
		{
			Label:            "(buffalo.validate.rules).double",
			Kind:             CompletionItemKindProperty,
			Detail:           "Double validation",
			InsertText:       "(buffalo.validate.rules).double = {\n  ${1|gte,gt,lte,lt|}: ${2:value}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Validation for double fields.\n\n```protobuf\ndouble latitude = 1 [(buffalo.validate.rules).double = {\n  gte: -90.0\n  lte: 90.0\n}];\n```",
			},
		},
		{
			Label:            "(buffalo.validate.rules).timestamp",
			Kind:             CompletionItemKindProperty,
			Detail:           "Timestamp validation",
			InsertText:       "(buffalo.validate.rules).timestamp = {\n  ${1|gt_now,lt_now,within_seconds|}: ${2:value}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Validation for timestamp fields.\n\n```protobuf\ngoogle.protobuf.Timestamp expires_at = 1 [(buffalo.validate.rules).timestamp = {\n  gt_now: true\n}];\n```",
			},
		},
		{
			Label:            "(buffalo.validate.rules).enum",
			Kind:             CompletionItemKindProperty,
			Detail:           "Enum validation",
			InsertText:       "(buffalo.validate.rules).enum = {\n  defined_only: true\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Validation for enum fields.\n\n```protobuf\nStatus status = 1 [(buffalo.validate.rules).enum = {\n  defined_only: true\n}];\n```",
			},
		},
		{
			Label:            "(buffalo.validate.rules).bytes",
			Kind:             CompletionItemKindProperty,
			Detail:           "Bytes validation",
			InsertText:       "(buffalo.validate.rules).bytes = {\n  min_len: ${1:1}\n  max_len: ${2:1048576}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Validation for bytes fields.\n\n```protobuf\nbytes data = 1 [(buffalo.validate.rules).bytes = {\n  min_len: 1\n  max_len: 1048576  // 1MB\n}];\n```",
			},
		},
		{
			Label:            "(buffalo.validate.rules).map",
			Kind:             CompletionItemKindProperty,
			Detail:           "Map validation",
			InsertText:       "(buffalo.validate.rules).map = {\n  min_pairs: ${1:1}\n  max_pairs: ${2:100}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Validation for map fields.\n\n```protobuf\nmap<string, string> metadata = 1 [(buffalo.validate.rules).map = {\n  max_pairs: 50\n}];\n```",
			},
		},
		// gRPC annotations
		{
			Label:            "google.api.http",
			Kind:             CompletionItemKindProperty,
			Detail:           "HTTP/REST mapping",
			InsertText:       "(google.api.http) = {\n  ${1|get,post,put,patch,delete|}: \"${2:/api/v1/resource}\"\n  ${3:body: \"*\"}\n}",
			InsertTextFormat: InsertTextFormatSnippet,
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "Map gRPC method to HTTP/REST endpoint.\n\n```protobuf\nrpc GetUser(GetUserRequest) returns (User) {\n  option (google.api.http) = {\n    get: \"/api/v1/users/{id}\"\n  };\n}\n```",
			},
		},
		// Deprecated
		{
			Label:            "deprecated",
			Kind:             CompletionItemKindProperty,
			Detail:           "Mark as deprecated",
			InsertText:       "deprecated = true",
			InsertTextFormat: InsertTextFormatPlainText,
		},
	}
}

// optionValueCompletions returns completions for option values.
func optionValueCompletions() []CompletionItem {
	return []CompletionItem{
		{Label: "true", Kind: CompletionItemKindValue},
		{Label: "false", Kind: CompletionItemKindValue},
	}
}

// commonImportCompletions returns completions for common imports.
func commonImportCompletions() []CompletionItem {
	return []CompletionItem{
		{
			Label:      "google/protobuf/empty.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "Empty message",
			InsertText: "google/protobuf/empty.proto",
		},
		{
			Label:      "google/protobuf/timestamp.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "Timestamp type",
			InsertText: "google/protobuf/timestamp.proto",
		},
		{
			Label:      "google/protobuf/duration.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "Duration type",
			InsertText: "google/protobuf/duration.proto",
		},
		{
			Label:      "google/protobuf/any.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "Any type",
			InsertText: "google/protobuf/any.proto",
		},
		{
			Label:      "google/protobuf/struct.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "Struct, Value, ListValue",
			InsertText: "google/protobuf/struct.proto",
		},
		{
			Label:      "google/protobuf/wrappers.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "Wrapper types",
			InsertText: "google/protobuf/wrappers.proto",
		},
		{
			Label:      "google/protobuf/field_mask.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "FieldMask type",
			InsertText: "google/protobuf/field_mask.proto",
		},
		{
			Label:      "google/api/annotations.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "HTTP annotations",
			InsertText: "google/api/annotations.proto",
		},
		// Buffalo embedded protos
		{
			Label:      "buffalo/validate/validate.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "Buffalo validation rules (embedded)",
			InsertText: "buffalo/validate/validate.proto",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "**Buffalo Validation**\n\nField validation rules for protobuf messages.\n\n```protobuf\nimport \"buffalo/validate/validate.proto\";\n\nmessage User {\n  string email = 1 [(buffalo.validate.rules).string = {\n    email: true\n    max_len: 255\n  }];\n}\n```\n\n✅ Embedded in Buffalo binary - no external dependency needed.",
			},
		},
		{
			Label:      "buffalo/validate.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "Buffalo validation (short path)",
			InsertText: "buffalo/validate.proto",
		},
		{
			Label:      "buffalo/permissions.proto",
			Kind:       CompletionItemKindFile,
			Detail:     "Buffalo permissions",
			InsertText: "buffalo/permissions.proto",
			Documentation: MarkupContent{
				Kind:  MarkupKindMarkdown,
				Value: "**Buffalo Permissions**\n\nRBAC/ABAC permission annotations for gRPC services.\n\n```protobuf\nimport \"buffalo/permissions.proto\";\n\nservice UserService {\n  option (buffalo.permissions.resource) = \"users\";\n  \n  rpc DeleteUser(DeleteUserRequest) returns (Empty) {\n    option (buffalo.permissions) = {\n      action: \"delete\"\n      roles: [\"admin\"]\n      audit_log: true\n    };\n  };\n}\n```",
			},
		},
	}
}

// getScalarTypeDoc returns documentation for scalar types.
func getScalarTypeDoc(typeName string) string {
	docs := map[string]string{
		"double":   "**double** - 64-bit IEEE 754 floating point\n\n- Default: `0.0`\n- Go: `float64`\n- Python: `float`\n- C++: `double`",
		"float":    "**float** - 32-bit IEEE 754 floating point\n\n- Default: `0.0`\n- Go: `float32`\n- Python: `float`\n- C++: `float`",
		"int32":    "**int32** - 32-bit signed integer (varint)\n\n- Default: `0`\n- Go: `int32`\n- Python: `int`\n- C++: `int32`\n\n⚠️ Inefficient for negative numbers, use `sint32` instead.",
		"int64":    "**int64** - 64-bit signed integer (varint)\n\n- Default: `0`\n- Go: `int64`\n- Python: `int`\n- C++: `int64`\n\n⚠️ Inefficient for negative numbers, use `sint64` instead.",
		"uint32":   "**uint32** - 32-bit unsigned integer (varint)\n\n- Default: `0`\n- Go: `uint32`\n- Python: `int`\n- C++: `uint32`",
		"uint64":   "**uint64** - 64-bit unsigned integer (varint)\n\n- Default: `0`\n- Go: `uint64`\n- Python: `int`\n- C++: `uint64`",
		"sint32":   "**sint32** - 32-bit signed integer (ZigZag + varint)\n\n- Default: `0`\n- Go: `int32`\n- Python: `int`\n- C++: `int32`\n\n✅ More efficient than `int32` for negative values.",
		"sint64":   "**sint64** - 64-bit signed integer (ZigZag + varint)\n\n- Default: `0`\n- Go: `int64`\n- Python: `int`\n- C++: `int64`\n\n✅ More efficient than `int64` for negative values.",
		"fixed32":  "**fixed32** - 32-bit unsigned (fixed 4 bytes)\n\n- Default: `0`\n- Go: `uint32`\n- Python: `int`\n- C++: `uint32`\n\n✅ More efficient than `uint32` when values > 2²⁸.",
		"fixed64":  "**fixed64** - 64-bit unsigned (fixed 8 bytes)\n\n- Default: `0`\n- Go: `uint64`\n- Python: `int`\n- C++: `uint64`\n\n✅ More efficient than `uint64` when values > 2⁵⁶.",
		"sfixed32": "**sfixed32** - 32-bit signed (fixed 4 bytes)\n\n- Default: `0`\n- Go: `int32`\n- Python: `int`\n- C++: `int32`",
		"sfixed64": "**sfixed64** - 64-bit signed (fixed 8 bytes)\n\n- Default: `0`\n- Go: `int64`\n- Python: `int`\n- C++: `int64`",
		"bool":     "**bool** - Boolean value\n\n- Default: `false`\n- Go: `bool`\n- Python: `bool`\n- C++: `bool`",
		"string":   "**string** - UTF-8 encoded string (max 2GB)\n\n- Default: `\"\"`\n- Go: `string`\n- Python: `str`\n- C++: `std::string`\n\n⚠️ Must be valid UTF-8 or 7-bit ASCII.",
		"bytes":    "**bytes** - Arbitrary byte sequence (max 2GB)\n\n- Default: empty bytes\n- Go: `[]byte`\n- Python: `bytes`\n- C++: `std::string`",
	}
	return docs[typeName]
}

// getKeywordDoc returns documentation for keywords.
func getKeywordDoc(keyword string) string {
	docs := map[string]string{
		"syntax":     "**syntax** - Specifies the protobuf syntax version.\n\n```protobuf\nsyntax = \"proto3\";\n```\n\nMust be the first non-comment line in the file.",
		"package":    "**package** - Declares the package namespace.\n\n```protobuf\npackage mycompany.myproject;\n```\n\nUsed for namespacing and code generation.",
		"import":     "**import** - Imports definitions from another proto file.\n\n```protobuf\nimport \"google/protobuf/timestamp.proto\";\nimport public \"other.proto\";  // transitive\nimport weak \"optional.proto\";   // optional\n```",
		"option":     "**option** - Sets a file-level, message-level, or field-level option.\n\n```protobuf\noption java_package = \"com.example\";\noption go_package = \"github.com/example/pkg\";\n```",
		"message":    "**message** - Defines a structured data type.\n\n```protobuf\nmessage User {\n  string name = 1;\n  int32 age = 2;\n  repeated string tags = 3;\n}\n```",
		"service":    "**service** - Defines a gRPC service with RPC methods.\n\n```protobuf\nservice UserService {\n  rpc GetUser(GetUserRequest) returns (User);\n  rpc ListUsers(ListRequest) returns (stream User);\n}\n```",
		"enum":       "**enum** - Defines an enumeration type.\n\n```protobuf\nenum Status {\n  STATUS_UNSPECIFIED = 0;  // must have zero value\n  STATUS_ACTIVE = 1;\n  STATUS_INACTIVE = 2;\n}\n```\n\n⚠️ First value must be 0 in proto3.",
		"rpc":        "**rpc** - Defines an RPC method in a service.\n\n```protobuf\nrpc GetUser(GetUserRequest) returns (User);\nrpc ListUsers(ListRequest) returns (stream User);  // server streaming\nrpc Upload(stream File) returns (Result);  // client streaming\nrpc Chat(stream Msg) returns (stream Msg);  // bidirectional\n```",
		"returns":    "**returns** - Specifies the return type of an RPC method.\n\n```protobuf\nrpc Method(Request) returns (Response);\nrpc Method(Request) returns (stream Response);  // streaming\n```",
		"stream":     "**stream** - Indicates a streaming parameter or return type.\n\n- Server streaming: `returns (stream Response)`\n- Client streaming: `(stream Request)`\n- Bidirectional: both",
		"repeated":   "**repeated** - Declares a field as a list/array.\n\n```protobuf\nrepeated string tags = 1;  // List of strings\nrepeated User users = 2;   // List of messages\n```\n\nDefault is empty list.",
		"optional":   "**optional** - Declares a field as optional with presence tracking.\n\n```protobuf\noptional string nickname = 1;\n```\n\nIn proto3, enables `has_*` methods for scalar fields.",
		"required":   "**required** (proto2 only) - Field must be set.\n\n⚠️ Deprecated: causes compatibility issues. Not available in proto3.",
		"oneof":      "**oneof** - Only one field in the group can be set.\n\n```protobuf\noneof result {\n  Error error = 1;\n  Data data = 2;\n}\n```\n\nSetting one field clears others in the group.",
		"map":        "**map** - Declares a map/dictionary field.\n\n```protobuf\nmap<string, int32> scores = 1;\nmap<int64, User> users = 2;\n```\n\n⚠️ Key type must be integral or string.",
		"reserved":   "**reserved** - Reserves field numbers or names.\n\n```protobuf\nreserved 2, 15, 9 to 11;\nreserved \"old_field\", \"deprecated_field\";\n```\n\nPrevents reuse of removed fields.",
		"extensions": "**extensions** (proto2) - Declares extension ranges.\n\n```protobuf\nextensions 100 to 199;\n```",
		"extend":     "**extend** - Extends a message with additional fields.\n\n```protobuf\nextend google.protobuf.FieldOptions {\n  optional string my_option = 51234;\n}\n```",
	}
	return docs[strings.ToLower(keyword)]
}
