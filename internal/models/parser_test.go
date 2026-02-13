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
