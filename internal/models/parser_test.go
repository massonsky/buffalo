package models

import (
	"testing"
)

func TestParseORMPlugin(t *testing.T) {
	tests := []struct {
		input   string
		name    string
		version string
		isNone  bool
	}{
		{"", "None", "", true},
		{"default", "None", "", true},
		{"None", "None", "", true},
		{"pydantic", "pydantic", "", false},
		{"pydantic@2.0", "pydantic", "2.0", false},
		{"sqlalchemy@2.0.25", "sqlalchemy", "2.0.25", false},
		{"gorm", "gorm", "", false},
		{"diesel@2.1", "diesel", "2.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := ParseORMPlugin(tt.input)
			if p.Name != tt.name {
				t.Errorf("ParseORMPlugin(%q).Name = %q, want %q", tt.input, p.Name, tt.name)
			}
			if p.Version != tt.version {
				t.Errorf("ParseORMPlugin(%q).Version = %q, want %q", tt.input, p.Version, tt.version)
			}
			if p.IsNone() != tt.isNone {
				t.Errorf("ParseORMPlugin(%q).IsNone() = %v, want %v", tt.input, p.IsNone(), tt.isNone)
			}
		})
	}
}

func TestParseORMPluginString(t *testing.T) {
	tests := []struct {
		plugin ORMPlugin
		want   string
	}{
		{ORMPlugin{Name: "pydantic", Version: "2.0"}, "pydantic@2.0"},
		{ORMPlugin{Name: "gorm"}, "gorm"},
		{ORMPlugin{Name: "None"}, "None"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.plugin.String()
			if got != tt.want {
				t.Errorf("ORMPlugin.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractModels(t *testing.T) {
	proto := `
syntax = "proto3";
package myservice;

import "buffalo/models/models.proto";

message User {
  option (buffalo.models.model) = {
    name: "User"
    description: "System user"
    tags: ["core", "auth"]
    timestamps: true
    soft_delete: true
  };

  string id = 1 [(buffalo.models.field) = {
    primary_key: true
    description: "Unique identifier"
    auto_generate: true
  }];

  string email = 2 [(buffalo.models.field) = {
    unique: true
    max_length: 255
    example: "user@example.com"
    visibility: PUBLIC
  }];

  string name = 3 [(buffalo.models.field) = {
    max_length: 100
    nullable: false
  }];

  int32 age = 4 [(buffalo.models.field) = {
    nullable: true
    comment: "user age"
  }];

  string password_hash = 5 [(buffalo.models.field) = {
    behavior: WRITEONLY
    sensitive: true
    visibility: PRIVATE
  }];

  string status = 6 [(buffalo.models.field) = {
    default_value: "active"
    max_length: 20
    visibility: INTERNAL
  }];

  string computed_display = 7 [(buffalo.models.field) = {
    behavior: COMPUTED
    db_ignore: true
  }];
}

message NoAnnotation {
  string id = 1;
  string name = 2;
}
`

	models, err := ExtractModels(proto, "test.proto")
	if err != nil {
		t.Fatalf("ExtractModels failed: %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	m := models[0]

	// Model-level checks
	if m.MessageName != "User" {
		t.Errorf("MessageName = %q, want %q", m.MessageName, "User")
	}
	if m.Package != "myservice" {
		t.Errorf("Package = %q, want %q", m.Package, "myservice")
	}
	if m.Name != "User" {
		t.Errorf("Name = %q, want %q", m.Name, "User")
	}
	if m.Description != "System user" {
		t.Errorf("Description = %q, want %q", m.Description, "System user")
	}
	if !m.Timestamps {
		t.Error("Timestamps should be true")
	}
	if !m.SoftDelete {
		t.Error("SoftDelete should be true")
	}
	if len(m.Tags) != 2 || m.Tags[0] != "core" || m.Tags[1] != "auth" {
		t.Errorf("Tags = %v, want [core auth]", m.Tags)
	}

	if len(m.Fields) != 7 {
		t.Fatalf("expected 7 fields, got %d", len(m.Fields))
	}

	// Field checks
	idField := m.Fields[0]
	if idField.Name != "id" || !idField.PrimaryKey || !idField.AutoGenerate {
		t.Errorf("id field: Name=%q PK=%v AutoGen=%v", idField.Name, idField.PrimaryKey, idField.AutoGenerate)
	}

	emailField := m.Fields[1]
	if emailField.Name != "email" || !emailField.Unique || emailField.MaxLength != 255 {
		t.Errorf("email field: Name=%q Unique=%v MaxLen=%d", emailField.Name, emailField.Unique, emailField.MaxLength)
	}
	if emailField.Visibility != VisibilityPublic {
		t.Errorf("email visibility = %v, want PUBLIC", emailField.Visibility)
	}
	if emailField.Example != "user@example.com" {
		t.Errorf("email example = %q, want %q", emailField.Example, "user@example.com")
	}

	pwdField := m.Fields[4]
	if pwdField.Name != "password_hash" || pwdField.Behavior != BehaviorWriteOnly || !pwdField.Sensitive {
		t.Errorf("password_hash: Name=%q Behavior=%v Sensitive=%v", pwdField.Name, pwdField.Behavior, pwdField.Sensitive)
	}
	if pwdField.Visibility != VisibilityPrivate {
		t.Errorf("password_hash visibility = %v, want PRIVATE", pwdField.Visibility)
	}

	statusField := m.Fields[5]
	if statusField.Visibility != VisibilityInternal {
		t.Errorf("status visibility = %v, want INTERNAL", statusField.Visibility)
	}

	computedField := m.Fields[6]
	if computedField.Behavior != BehaviorComputed || !computedField.DBIgnore {
		t.Errorf("computed_display: Behavior=%v DBIgnore=%v", computedField.Behavior, computedField.DBIgnore)
	}
}

func TestFieldDefHelpers(t *testing.T) {
	t.Run("IsSerializable", func(t *testing.T) {
		tests := []struct {
			name   string
			field  FieldDef
			expect bool
		}{
			{"normal", FieldDef{Name: "x"}, true},
			{"ignore", FieldDef{Name: "x", Ignore: true}, false},
			{"api_ignore", FieldDef{Name: "x", APIIgnore: true}, false},
			{"private", FieldDef{Name: "x", Visibility: VisibilityPrivate}, false},
			{"writeonly", FieldDef{Name: "x", Behavior: BehaviorWriteOnly}, false},
			{"internal", FieldDef{Name: "x", Visibility: VisibilityInternal}, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.field.IsSerializable() != tt.expect {
					t.Errorf("IsSerializable() = %v, want %v", tt.field.IsSerializable(), tt.expect)
				}
			})
		}
	})

	t.Run("IsPersistable", func(t *testing.T) {
		tests := []struct {
			name   string
			field  FieldDef
			expect bool
		}{
			{"normal", FieldDef{Name: "x"}, true},
			{"ignore", FieldDef{Name: "x", Ignore: true}, false},
			{"db_ignore", FieldDef{Name: "x", DBIgnore: true}, false},
			{"virtual", FieldDef{Name: "x", Behavior: BehaviorVirtual}, false},
			{"computed", FieldDef{Name: "x", Behavior: BehaviorComputed}, false},
			{"readonly", FieldDef{Name: "x", Behavior: BehaviorReadOnly}, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if tt.field.IsPersistable() != tt.expect {
					t.Errorf("IsPersistable() = %v, want %v", tt.field.IsPersistable(), tt.expect)
				}
			})
		}
	})
}

func TestEffectiveJSONName(t *testing.T) {
	f := FieldDef{Name: "my_field"}
	if f.EffectiveJSONName() != "my_field" {
		t.Errorf("expected my_field, got %s", f.EffectiveJSONName())
	}

	f.JSONName = "myField"
	if f.EffectiveJSONName() != "myField" {
		t.Errorf("expected myField, got %s", f.EffectiveJSONName())
	}
}

func TestVisibilityString(t *testing.T) {
	tests := map[FieldVisibility]string{
		VisibilityDefault:   "default",
		VisibilityPublic:    "public",
		VisibilityInternal:  "internal",
		VisibilityExternal:  "external",
		VisibilityPrivate:   "private",
		VisibilityProtected: "protected",
	}
	for v, want := range tests {
		if v.String() != want {
			t.Errorf("FieldVisibility(%d).String() = %q, want %q", v, v.String(), want)
		}
	}
}

func TestBehaviorString(t *testing.T) {
	tests := map[FieldBehavior]string{
		BehaviorDefault:    "default",
		BehaviorReadOnly:   "readonly",
		BehaviorWriteOnly:  "writeonly",
		BehaviorImmutable:  "immutable",
		BehaviorComputed:   "computed",
		BehaviorVirtual:    "virtual",
		BehaviorOutputOnly: "output_only",
		BehaviorInputOnly:  "input_only",
	}
	for b, want := range tests {
		if b.String() != want {
			t.Errorf("FieldBehavior(%d).String() = %q, want %q", b, b.String(), want)
		}
	}
}

// ══════════════════════════════════════════════════════════════════
//  ExtractAllMessages tests
// ══════════════════════════════════════════════════════════════════

func TestExtractAllMessages_Basic(t *testing.T) {
	proto := `
syntax = "proto3";
package araviec.common.v1;
option go_package = "araviec/go/common/v1;commonv1";

// Разрешение экрана или матрицы
message Resolution {
  int32 width = 1;
  int32 height = 2;
}

// Точка в 2D пространстве
message Point {
  float x = 1;
  float y = 2;
}

// Прямоугольник (ROI)
message Rectangle {
  Point top_left = 1;
  Resolution size = 2;
}

// Полигон
message Polygon {
  repeated Point vertices = 1;
}
`
	models, err := ExtractAllMessages(proto, "math.proto")
	if err != nil {
		t.Fatalf("ExtractAllMessages failed: %v", err)
	}

	if len(models) != 4 {
		t.Fatalf("expected 4 models, got %d", len(models))
	}

	// Check message names
	names := make(map[string]bool)
	for _, m := range models {
		names[m.MessageName] = true
	}
	for _, expected := range []string{"Resolution", "Point", "Rectangle", "Polygon"} {
		if !names[expected] {
			t.Errorf("missing model %q", expected)
		}
	}

	// Check package
	if models[0].Package != "araviec.common.v1" {
		t.Errorf("Package = %q, want %q", models[0].Package, "araviec.common.v1")
	}

	// Check Resolution fields
	var resolution ModelDef
	for _, m := range models {
		if m.MessageName == "Resolution" {
			resolution = m
			break
		}
	}
	if len(resolution.Fields) != 2 {
		t.Fatalf("Resolution: expected 2 fields, got %d", len(resolution.Fields))
	}
	if resolution.Fields[0].Name != "width" || resolution.Fields[0].ProtoType != "int32" {
		t.Errorf("Resolution.width: Name=%q Type=%q", resolution.Fields[0].Name, resolution.Fields[0].ProtoType)
	}
	if resolution.Description == "" {
		t.Error("Resolution should have a description from comment")
	}

	// Check Rectangle has message-type fields
	var rect ModelDef
	for _, m := range models {
		if m.MessageName == "Rectangle" {
			rect = m
			break
		}
	}
	if len(rect.Fields) != 2 {
		t.Fatalf("Rectangle: expected 2 fields, got %d", len(rect.Fields))
	}
	if rect.Fields[0].ProtoType != "Point" {
		t.Errorf("Rectangle.top_left type = %q, want %q", rect.Fields[0].ProtoType, "Point")
	}

	// Check Polygon has repeated field
	var poly ModelDef
	for _, m := range models {
		if m.MessageName == "Polygon" {
			poly = m
			break
		}
	}
	if len(poly.Fields) != 1 {
		t.Fatalf("Polygon: expected 1 field, got %d", len(poly.Fields))
	}
	if !poly.Fields[0].Repeated {
		t.Error("Polygon.vertices should be repeated")
	}
}

func TestExtractAllMessages_MapField(t *testing.T) {
	proto := `
syntax = "proto3";
package test;

message Settings {
  string source_id = 1;
  map<string, string> settings = 2;
  map<string, int32> counts = 3;
}
`
	models, err := ExtractAllMessages(proto, "settings.proto")
	if err != nil {
		t.Fatalf("ExtractAllMessages failed: %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	m := models[0]
	if len(m.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(m.Fields))
	}

	// Check map field "settings"
	var settingsField, countsField FieldDef
	for _, f := range m.Fields {
		if f.Name == "settings" {
			settingsField = f
		}
		if f.Name == "counts" {
			countsField = f
		}
	}

	if !settingsField.IsMap {
		t.Error("settings field should be IsMap=true")
	}
	if settingsField.MapKeyType != "string" {
		t.Errorf("settings.MapKeyType = %q, want %q", settingsField.MapKeyType, "string")
	}
	if settingsField.MapValueType != "string" {
		t.Errorf("settings.MapValueType = %q, want %q", settingsField.MapValueType, "string")
	}

	if !countsField.IsMap {
		t.Error("counts field should be IsMap=true")
	}
	if countsField.MapKeyType != "string" || countsField.MapValueType != "int32" {
		t.Errorf("counts map types: <%s, %s>", countsField.MapKeyType, countsField.MapValueType)
	}
}

func TestExtractAllMessages_NestedEnum(t *testing.T) {
	proto := `
syntax = "proto3";
package test;

message MultimediaSource {
  string source_id = 1;
  SourceStatus status = 2;

  enum SourceStatus {
    SOURCE_STATUS_UNSPECIFIED = 0;
    SOURCE_STATUS_ACTIVE = 1;
    SOURCE_STATUS_INACTIVE = 2;
    SOURCE_STATUS_ERROR = 3;
  }
}
`
	models, err := ExtractAllMessages(proto, "source.proto")
	if err != nil {
		t.Fatalf("ExtractAllMessages failed: %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	m := models[0]

	// Check enums were extracted
	if len(m.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(m.Enums))
	}
	if m.Enums[0].Name != "SourceStatus" {
		t.Errorf("enum name = %q, want %q", m.Enums[0].Name, "SourceStatus")
	}
	if len(m.Enums[0].Values) != 4 {
		t.Errorf("expected 4 enum values, got %d", len(m.Enums[0].Values))
	}

	// Enum field should be resolved to int32
	var statusField FieldDef
	for _, f := range m.Fields {
		if f.Name == "status" {
			statusField = f
			break
		}
	}
	if statusField.ProtoType != "int32" {
		t.Errorf("status field type = %q, want %q (enum -> int32)", statusField.ProtoType, "int32")
	}
}

func TestExtractAllMessages_WithAnnotations(t *testing.T) {
	proto := `
syntax = "proto3";
package test;

import "buffalo/models/models.proto";

message User {
  option (buffalo.models.model) = {
    name: "User"
    timestamps: true
  };

  string id = 1 [(buffalo.models.field) = {
    primary_key: true
  }];
  string email = 2;
}

message Profile {
  string avatar_url = 1;
  string bio = 2;
}
`
	models, err := ExtractAllMessages(proto, "user.proto")
	if err != nil {
		t.Fatalf("ExtractAllMessages failed: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models (User + Profile), got %d", len(models))
	}

	// User should have annotations applied
	var user, profile ModelDef
	for _, m := range models {
		if m.MessageName == "User" {
			user = m
		}
		if m.MessageName == "Profile" {
			profile = m
		}
	}

	if user.Name != "User" {
		t.Errorf("User.Name = %q, want %q", user.Name, "User")
	}
	if !user.Timestamps {
		t.Error("User.Timestamps should be true")
	}
	if !user.Fields[0].PrimaryKey {
		t.Error("User.id should be primary_key")
	}

	// Profile should be auto-derived (no annotations)
	if profile.Name != "" {
		t.Errorf("Profile.Name should be empty (auto-derived), got %q", profile.Name)
	}
	if len(profile.Fields) != 2 {
		t.Errorf("Profile should have 2 fields, got %d", len(profile.Fields))
	}
}

func TestExtractAllMessages_ServiceSkipped(t *testing.T) {
	proto := `
syntax = "proto3";
package test;

service MyService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}

message GetUserRequest {
  string user_id = 1;
}

message GetUserResponse {
  string name = 1;
  string email = 2;
}
`
	models, err := ExtractAllMessages(proto, "service.proto")
	if err != nil {
		t.Fatalf("ExtractAllMessages failed: %v", err)
	}

	// Service should be skipped, only messages extracted
	if len(models) != 2 {
		t.Fatalf("expected 2 models (request + response), got %d", len(models))
	}

	names := make(map[string]bool)
	for _, m := range models {
		names[m.MessageName] = true
	}
	if names["MyService"] {
		t.Error("service MyService should not be extracted as a model")
	}
}

func TestExtractAllMessages_CrossPackageTypes(t *testing.T) {
	proto := `
syntax = "proto3";
package araviec.multimedia.v1;
import "araviec/common/v1/math.proto";

message VideoOptions {
  araviec.common.v1.Resolution resolution = 1;
  int32 fps = 2;
  int32 bitrate = 3;
  string codec = 4;
}
`
	models, err := ExtractAllMessages(proto, "multimedia.proto")
	if err != nil {
		t.Fatalf("ExtractAllMessages failed: %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	m := models[0]
	if len(m.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(m.Fields))
	}

	// Cross-package type reference should be preserved
	if m.Fields[0].ProtoType != "araviec.common.v1.Resolution" {
		t.Errorf("resolution field type = %q, want %q", m.Fields[0].ProtoType, "araviec.common.v1.Resolution")
	}
}

func TestFieldTypeHelpers(t *testing.T) {
	t.Run("Go map field", func(t *testing.T) {
		f := FieldDef{Name: "settings", IsMap: true, MapKeyType: "string", MapValueType: "string"}
		got := fieldTypeGo(f)
		if got != "map[string]string" {
			t.Errorf("fieldTypeGo = %q, want %q", got, "map[string]string")
		}
	})

	t.Run("Go repeated field", func(t *testing.T) {
		f := FieldDef{Name: "tags", ProtoType: "string", Repeated: true}
		got := fieldTypeGo(f)
		if got != "[]string" {
			t.Errorf("fieldTypeGo = %q, want %q", got, "[]string")
		}
	})

	t.Run("Python map field", func(t *testing.T) {
		f := FieldDef{Name: "settings", IsMap: true, MapKeyType: "string", MapValueType: "int32"}
		got := fieldTypePython(f)
		if got != "Dict[str, int]" {
			t.Errorf("fieldTypePython = %q, want %q", got, "Dict[str, int]")
		}
	})

	t.Run("Rust map field", func(t *testing.T) {
		f := FieldDef{Name: "settings", IsMap: true, MapKeyType: "string", MapValueType: "string"}
		got := fieldTypeRust(f)
		if got != "std::collections::HashMap<String, String>" {
			t.Errorf("fieldTypeRust = %q, want %q", got, "std::collections::HashMap<String, String>")
		}
	})

	t.Run("Cpp map field", func(t *testing.T) {
		f := FieldDef{Name: "settings", IsMap: true, MapKeyType: "string", MapValueType: "string"}
		got := fieldTypeCpp(f)
		if got != "std::map<std::string, std::string>" {
			t.Errorf("fieldTypeCpp = %q, want %q", got, "std::map<std::string, std::string>")
		}
	})

	t.Run("stripPackagePrefix", func(t *testing.T) {
		if s := stripPackagePrefix("araviec.common.v1.Resolution"); s != "Resolution" {
			t.Errorf("stripPackagePrefix = %q, want %q", s, "Resolution")
		}
		if s := stripPackagePrefix("Resolution"); s != "Resolution" {
			t.Errorf("stripPackagePrefix = %q, want %q", s, "Resolution")
		}
	})
}

func TestExtractAllMessages_FullMultimedia(t *testing.T) {
	// Test with a realistic proto file similar to the user's endpoint service
	proto := `
syntax = "proto3";
package araviec.endpoints.v1;
option go_package = "araviec/go/endpoints/v1/service;v1";

service EndpointService {
    rpc UpdateConfig(UpdateConfigRequest) returns (UpdateConfigResponse);
    rpc GetStats(GetStatsRequest) returns (GetStatsResponse);
}

message UpdateConfigRequest {
    string remote_ip = 1;
    int32 remote_port = 2;
}

message UpdateConfigResponse {
    bool success = 1;
    string message = 2;
}

message GetStatsRequest {}

message GetStatsResponse {
    uint64 uplink_packets = 1;
    uint64 downlink_packets = 2;
    uint64 dropped_packets = 3;
    bool nats_connected = 4;
}

// Метрики системы
message SystemMetrics {
    double cpu_percent = 1;
    uint64 memory_used_mb = 2;
    uint64 memory_total_mb = 3;
    uint32 goroutines = 4;
    uint64 uptime_seconds = 5;
}

// Ошибки
message ErrorMetrics {
    uint64 total_errors = 1;
    double error_rate = 2;
    map<string, uint64> errors_by_type = 3;
    repeated ErrorEntry recent_errors = 4;
}

message ErrorEntry {
    string type = 1;
    string message = 2;
    int64 timestamp = 3;
    uint64 count = 4;
}
`

	models, err := ExtractAllMessages(proto, "endpoint.proto")
	if err != nil {
		t.Fatalf("ExtractAllMessages failed: %v", err)
	}

	// Count: UpdateConfigRequest, UpdateConfigResponse, GetStatsRequest,
	//        GetStatsResponse, SystemMetrics, ErrorMetrics, ErrorEntry = 7
	if len(models) != 7 {
		t.Fatalf("expected 7 models, got %d", len(models))
	}

	// Verify ErrorMetrics has map and repeated fields
	var errMetrics ModelDef
	for _, m := range models {
		if m.MessageName == "ErrorMetrics" {
			errMetrics = m
			break
		}
	}
	if len(errMetrics.Fields) != 4 {
		t.Fatalf("ErrorMetrics: expected 4 fields, got %d", len(errMetrics.Fields))
	}

	// map<string, uint64> errors_by_type
	var mapField FieldDef
	for _, f := range errMetrics.Fields {
		if f.Name == "errors_by_type" {
			mapField = f
			break
		}
	}
	if !mapField.IsMap {
		t.Error("errors_by_type should be IsMap=true")
	}
	if mapField.MapKeyType != "string" || mapField.MapValueType != "uint64" {
		t.Errorf("errors_by_type map types: <%s, %s>", mapField.MapKeyType, mapField.MapValueType)
	}

	// repeated ErrorEntry recent_errors
	var repeatedField FieldDef
	for _, f := range errMetrics.Fields {
		if f.Name == "recent_errors" {
			repeatedField = f
			break
		}
	}
	if !repeatedField.Repeated {
		t.Error("recent_errors should be Repeated=true")
	}
	if repeatedField.ProtoType != "ErrorEntry" {
		t.Errorf("recent_errors type = %q, want %q", repeatedField.ProtoType, "ErrorEntry")
	}

	// GetStatsRequest should have 0 fields (empty message)
	var emptyMsg ModelDef
	for _, m := range models {
		if m.MessageName == "GetStatsRequest" {
			emptyMsg = m
			break
		}
	}
	if len(emptyMsg.Fields) != 0 {
		t.Errorf("GetStatsRequest should have 0 fields, got %d", len(emptyMsg.Fields))
	}

	// SystemMetrics should have description from comment
	var sysMetrics ModelDef
	for _, m := range models {
		if m.MessageName == "SystemMetrics" {
			sysMetrics = m
			break
		}
	}
	if sysMetrics.Description == "" {
		t.Error("SystemMetrics should have description from comment")
	}
}
