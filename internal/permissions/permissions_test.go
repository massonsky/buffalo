package permissions

import (
	"context"
	"strings"
	"testing"
)

func TestParser_ParseFile(t *testing.T) {
	// Create a mock proto content for testing
	parser := NewParser()

	// Test with empty context
	t.Run("new parser", func(t *testing.T) {
		if parser == nil {
			t.Error("parser should not be nil")
		}
	})
}

func TestParser_ParseStringList(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{`"admin", "user"`, []string{"admin", "user"}},
		{`"read"`, []string{"read"}},
		{``, []string{}},
		{`"role1","role2","role3"`, []string{"role1", "role2", "role3"}},
	}

	for _, tc := range tests {
		result := parseStringList(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parseStringList(%q) = %v, expected %v", tc.input, result, tc.expected)
		}
	}
}

func TestAnalyzer_DefaultRules(t *testing.T) {
	analyzer := NewAnalyzer()

	if len(analyzer.rules) == 0 {
		t.Error("expected default rules to be loaded")
	}
}

func TestAnalyzer_Audit(t *testing.T) {
	analyzer := NewAnalyzer()
	ctx := context.Background()

	tests := []struct {
		name        string
		services    []*ServicePermissions
		expectCount int
		expectRule  string
	}{
		{
			name: "no roles or scopes",
			services: []*ServicePermissions{
				{
					Service:  "test.Service",
					Resource: "test",
					Methods: map[string]*Permission{
						"GetUser": {
							Action: "users:read",
							// No roles or scopes, not public
						},
					},
				},
			},
			expectCount: 1,
			expectRule:  "NO_ROLES_OR_SCOPES",
		},
		{
			name: "public endpoint allowed",
			services: []*ServicePermissions{
				{
					Service:  "test.Service",
					Resource: "test",
					Methods: map[string]*Permission{
						"GetPublic": {
							Action: "public:read",
							Public: true,
						},
					},
				},
			},
			expectCount: 1, // Only MISSING_RATE_LIMIT
			expectRule:  "MISSING_RATE_LIMIT",
		},
		{
			name: "admin without MFA",
			services: []*ServicePermissions{
				{
					Service:  "test.Service",
					Resource: "test",
					Methods: map[string]*Permission{
						"AdminAction": {
							Action: "admin:write",
							Roles:  []string{"admin"},
							// RequireMFA: false
						},
					},
				},
			},
			expectCount: 2, // ADMIN_NO_MFA and SENSITIVE_NO_AUDIT
		},
		{
			name: "wildcard role",
			services: []*ServicePermissions{
				{
					Service:  "test.Service",
					Resource: "test",
					Methods: map[string]*Permission{
						"AnyUser": {
							Action: "any:read",
							Roles:  []string{"*"},
						},
					},
				},
			},
			expectCount: 1,
			expectRule:  "WILDCARD_ROLE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := analyzer.Audit(ctx, tc.services)
			if len(issues) < tc.expectCount {
				t.Errorf("expected at least %d issues, got %d", tc.expectCount, len(issues))
			}

			if tc.expectRule != "" {
				found := false
				for _, issue := range issues {
					if issue.RuleID == tc.expectRule {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected rule %s to be triggered, issues: %v", tc.expectRule, issues)
				}
			}
		})
	}
}

func TestAnalyzer_Diff(t *testing.T) {
	analyzer := NewAnalyzer()
	ctx := context.Background()

	old := []*ServicePermissions{
		{
			Service: "test.Service",
			Methods: map[string]*Permission{
				"GetUser":    {Action: "users:read", Roles: []string{"user"}},
				"DeleteUser": {Action: "users:delete", Roles: []string{"admin"}},
			},
		},
	}

	new := []*ServicePermissions{
		{
			Service: "test.Service",
			Methods: map[string]*Permission{
				"GetUser":    {Action: "users:read", Roles: []string{"user", "guest"}}, // Modified
				"CreateUser": {Action: "users:create", Roles: []string{"admin"}},       // Added
				// DeleteUser removed
			},
		},
	}

	diffs := analyzer.Diff(ctx, old, new)

	// Check for expected diffs
	var added, modified, removed int
	for _, d := range diffs {
		switch d.Type {
		case DiffAdded:
			added++
		case DiffModified:
			modified++
		case DiffRemoved:
			removed++
		}
	}

	if added != 1 {
		t.Errorf("expected 1 added, got %d", added)
	}
	if modified != 1 {
		t.Errorf("expected 1 modified, got %d", modified)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
}

func TestAnalyzer_Summary(t *testing.T) {
	analyzer := NewAnalyzer()

	services := []*ServicePermissions{
		{
			Service: "test.Service1",
			Methods: map[string]*Permission{
				"Method1": {Roles: []string{"admin", "user"}},
				"Method2": {Scopes: []string{"read"}, Public: true},
			},
		},
		{
			Service: "test.Service2",
			Methods: map[string]*Permission{
				"Method3": {Roles: []string{"admin"}},
			},
		},
	}

	summary := analyzer.Summary(services)

	if summary.ServiceCount != 2 {
		t.Errorf("expected 2 services, got %d", summary.ServiceCount)
	}
	if summary.MethodCount != 3 {
		t.Errorf("expected 3 methods, got %d", summary.MethodCount)
	}
	if summary.PublicCount != 1 {
		t.Errorf("expected 1 public, got %d", summary.PublicCount)
	}
	if summary.ByRole["admin"] != 2 {
		t.Errorf("expected admin role count 2, got %d", summary.ByRole["admin"])
	}
}

func TestBuildMatrix(t *testing.T) {
	services := []*ServicePermissions{
		{
			Service: "test.Service",
			Methods: map[string]*Permission{
				"GetUser":    {Action: "users:read", Roles: []string{"user", "admin"}},
				"DeleteUser": {Action: "users:delete", Roles: []string{"admin"}, RequireMFA: true},
			},
		},
	}

	matrix := BuildMatrix(services)

	if len(matrix.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(matrix.Services))
	}
	if len(matrix.Roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(matrix.Roles))
	}

	// Check access
	key := "test.Service.GetUser"
	if access := matrix.Access[key]["role:user"]; access == nil || !access.Allowed {
		t.Error("expected user role to have access to GetUser")
	}

	key = "test.Service.DeleteUser"
	if access := matrix.Access[key]["role:admin"]; access == nil || !access.RequireMFA {
		t.Error("expected DeleteUser to require MFA")
	}
}

func TestMatrix_RenderText(t *testing.T) {
	services := []*ServicePermissions{
		{
			Service: "test.Service",
			Methods: map[string]*Permission{
				"GetUser": {Action: "users:read", Roles: []string{"admin"}},
			},
		},
	}

	matrix := BuildMatrix(services)
	text := matrix.RenderText()

	if !strings.Contains(text, "test.Service") {
		t.Error("expected output to contain service name")
	}
	if !strings.Contains(text, "GetUser") {
		t.Error("expected output to contain method name")
	}
	if !strings.Contains(text, "✓") {
		t.Error("expected output to contain checkmark for allowed access")
	}
}

func TestMatrix_RenderMarkdown(t *testing.T) {
	services := []*ServicePermissions{
		{
			Service: "test.Service",
			Methods: map[string]*Permission{
				"GetUser": {Action: "users:read", Roles: []string{"admin"}},
			},
		},
	}

	matrix := BuildMatrix(services)
	md := matrix.RenderMarkdown()

	if !strings.Contains(md, "# Permission Access Matrix") {
		t.Error("expected markdown header")
	}
	if !strings.Contains(md, "| Method |") {
		t.Error("expected markdown table")
	}
}

func TestMatrix_RenderHTML(t *testing.T) {
	services := []*ServicePermissions{
		{
			Service: "test.Service",
			Methods: map[string]*Permission{
				"GetUser": {Action: "users:read", Roles: []string{"admin"}},
			},
		},
	}

	matrix := BuildMatrix(services)
	html, err := matrix.RenderHTML()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("expected valid HTML")
	}
	if !strings.Contains(html, "Permission Access Matrix") {
		t.Error("expected HTML title")
	}
}

func TestGenerator_GenerateGo(t *testing.T) {
	gen := NewGenerator(GeneratorOptions{
		Framework:         "go",
		Package:           "testpkg",
		GenerateConstants: true,
	})

	services := []*ServicePermissions{
		{
			Service: "test.UserService",
			Methods: map[string]*Permission{
				"GetUser": {
					Action:     "users:read",
					Roles:      []string{"admin", "user"},
					RequireMFA: false,
				},
			},
		},
	}

	code, err := gen.Generate(services)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(code, "package testpkg") {
		t.Error("expected package declaration")
	}
	if !strings.Contains(code, "Permissions = map[string]Permission") {
		t.Error("expected Permissions map")
	}
	if !strings.Contains(code, "test.UserService.GetUser") {
		t.Error("expected permission entry")
	}
	if !strings.Contains(code, "ActionUsersRead") {
		t.Error("expected action constant")
	}
}

func TestGenerator_GenerateCasbin(t *testing.T) {
	gen := NewGenerator(GeneratorOptions{
		Framework: "casbin",
	})

	services := []*ServicePermissions{
		{
			Service: "test.Service",
			Methods: map[string]*Permission{
				"GetUser": {Action: "read", Roles: []string{"admin"}},
			},
		},
	}

	code, err := gen.Generate(services)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(code, "[request_definition]") {
		t.Error("expected Casbin model")
	}
	if !strings.Contains(code, "p, admin, test.Service.GetUser, read") {
		t.Error("expected Casbin policy")
	}
}

func TestGenerator_GenerateOPA(t *testing.T) {
	gen := NewGenerator(GeneratorOptions{
		Framework: "opa",
	})

	services := []*ServicePermissions{
		{
			Service: "test.Service",
			Methods: map[string]*Permission{
				"GetUser": {Action: "users:read", Roles: []string{"admin"}},
			},
		},
	}

	code, err := gen.Generate(services)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(code, "package buffalo.authz") {
		t.Error("expected OPA package")
	}
	if !strings.Contains(code, "permissions :=") {
		t.Error("expected permissions definition")
	}
}

func TestToConstName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users:read", "UsersRead"},
		{"admin.write", "AdminWrite"},
		{"super-admin", "SuperAdmin"},
		{"ROLE", "Role"},
	}

	for _, tc := range tests {
		result := toConstName(tc.input)
		if result != tc.expected {
			t.Errorf("toConstName(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
