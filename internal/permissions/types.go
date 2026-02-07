// Package permissions provides RBAC/ABAC permission management for protobuf services.
package permissions

// Permission represents a single permission rule for an RPC method.
type Permission struct {
	// Action is the permission action (e.g., "read", "write", "delete").
	Action string `json:"action" yaml:"action"`
	// Roles lists allowed roles.
	Roles []string `json:"roles,omitempty" yaml:"roles,omitempty"`
	// Scopes lists required OAuth scopes.
	Scopes []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	// Conditions are ABAC conditions.
	Conditions []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	// AllowSelf allows users to access their own resources.
	AllowSelf bool `json:"allow_self,omitempty" yaml:"allow_self,omitempty"`
	// RequireMFA requires multi-factor authentication.
	RequireMFA bool `json:"require_mfa,omitempty" yaml:"require_mfa,omitempty"`
	// AuditLog enables audit logging for this action.
	AuditLog bool `json:"audit_log,omitempty" yaml:"audit_log,omitempty"`
	// RateLimit sets rate limiting for this action.
	RateLimit *RateLimit `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
	// Public makes this endpoint public (no auth required).
	Public bool `json:"public,omitempty" yaml:"public,omitempty"`
	// OwnerField is the field name used for owner-based access checks.
	OwnerField string `json:"owner_field,omitempty" yaml:"owner_field,omitempty"`
	// RequireApproval requires manual approval for this action.
	RequireApproval bool `json:"require_approval,omitempty" yaml:"require_approval,omitempty"`
	// Condition is a CEL expression for conditional access.
	Condition string `json:"condition,omitempty" yaml:"condition,omitempty"`
	// FieldPermissions restricts access to specific message fields.
	FieldPermissions *FieldPermission `json:"field_permissions,omitempty" yaml:"field_permissions,omitempty"`
}

// FieldPermission restricts access to specific message fields by role.
type FieldPermission struct {
	// Field is the field name to restrict.
	Field string `json:"field" yaml:"field"`
	// Roles lists roles that can access this field.
	Roles []string `json:"roles" yaml:"roles"`
}

// Condition represents an ABAC condition.
type Condition struct {
	// Field is the field to check.
	Field string `json:"field" yaml:"field"`
	// Operator is the comparison operator.
	Operator ConditionOperator `json:"operator" yaml:"operator"`
	// Source is where to get the comparison value.
	Source string `json:"source" yaml:"source"`
	// Value is a static comparison value (if Source is not set).
	Value interface{} `json:"value,omitempty" yaml:"value,omitempty"`
}

// ConditionOperator is the type of condition comparison.
type ConditionOperator string

const (
	// OpEqual checks for equality.
	OpEqual ConditionOperator = "eq"
	// OpNotEqual checks for inequality.
	OpNotEqual ConditionOperator = "neq"
	// OpIn checks if value is in a list.
	OpIn ConditionOperator = "in"
	// OpNotIn checks if value is not in a list.
	OpNotIn ConditionOperator = "not_in"
	// OpContains checks if a list contains a value.
	OpContains ConditionOperator = "contains"
	// OpGreaterThan checks if value is greater.
	OpGreaterThan ConditionOperator = "gt"
	// OpLessThan checks if value is less.
	OpLessThan ConditionOperator = "lt"
	// OpExists checks if a field exists.
	OpExists ConditionOperator = "exists"
)

// RateLimit specifies rate limiting configuration.
type RateLimit struct {
	// Requests is the maximum number of requests.
	Requests int `json:"requests" yaml:"requests"`
	// Window is the time window (e.g., "1m", "1h").
	Window string `json:"window" yaml:"window"`
	// PerUser if true, rate limit is per user.
	PerUser bool `json:"per_user,omitempty" yaml:"per_user,omitempty"`
}

// ServicePermissions contains permissions for a service.
type ServicePermissions struct {
	// Service is the fully qualified service name.
	Service string `json:"service" yaml:"service"`
	// Resource is the resource name for this service.
	Resource string `json:"resource,omitempty" yaml:"resource,omitempty"`
	// DefaultRoles are default roles for all methods.
	DefaultRoles []string `json:"default_roles,omitempty" yaml:"default_roles,omitempty"`
	// Methods maps method names to permissions.
	Methods map[string]*Permission `json:"methods" yaml:"methods"`
}

// MethodPermission represents permissions for a specific RPC method.
type MethodPermission struct {
	// Service is the service name.
	Service string `json:"service" yaml:"service"`
	// Method is the method name.
	Method string `json:"method" yaml:"method"`
	// Permission is the permission configuration.
	Permission *Permission `json:"permission" yaml:"permission"`
}

// AccessLevel represents the level of access for matrix display.
type AccessLevel struct {
	// Allowed indicates if access is allowed.
	Allowed bool `json:"allowed" yaml:"allowed"`
	// RequireMFA indicates if MFA is required.
	RequireMFA bool `json:"require_mfa,omitempty" yaml:"require_mfa,omitempty"`
	// Conditions indicates if there are conditions.
	Conditions bool `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	// RateLimit indicates if there is rate limiting.
	RateLimit bool `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
}

// AuditEntry represents an audit log entry.
type AuditEntry struct {
	// Timestamp is when the action occurred.
	Timestamp string `json:"timestamp" yaml:"timestamp"`
	// Service is the service name.
	Service string `json:"service" yaml:"service"`
	// Method is the method name.
	Method string `json:"method" yaml:"method"`
	// UserID is the user who performed the action.
	UserID string `json:"user_id" yaml:"user_id"`
	// Roles are the user's roles.
	Roles []string `json:"roles" yaml:"roles"`
	// Action is the permission action.
	Action string `json:"action" yaml:"action"`
	// Resource is the resource being accessed.
	Resource string `json:"resource" yaml:"resource"`
	// ResourceID is the specific resource ID.
	ResourceID string `json:"resource_id,omitempty" yaml:"resource_id,omitempty"`
	// Allowed indicates if access was granted.
	Allowed bool `json:"allowed" yaml:"allowed"`
	// Reason explains why access was granted/denied.
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// PermissionDiff represents a difference between two permissions.
type PermissionDiff struct {
	// Type is the type of difference.
	Type DiffType `json:"type" yaml:"type"`
	// Service is the service name.
	Service string `json:"service" yaml:"service"`
	// Method is the method name.
	Method string `json:"method" yaml:"method"`
	// Old is the old permission (for removed/modified).
	Old *Permission `json:"old,omitempty" yaml:"old,omitempty"`
	// New is the new permission (for added/modified).
	New *Permission `json:"new,omitempty" yaml:"new,omitempty"`
}

// DiffType represents the type of permission change.
type DiffType string

const (
	// DiffAdded means a new permission was added.
	DiffAdded DiffType = "added"
	// DiffRemoved means a permission was removed.
	DiffRemoved DiffType = "removed"
	// DiffModified means a permission was modified.
	DiffModified DiffType = "modified"
)

// AuditIssue represents a permission audit issue.
type AuditIssue struct {
	// RuleID is the rule identifier.
	RuleID string `json:"rule_id" yaml:"rule_id"`
	// Severity is the issue severity.
	Severity IssueSeverity `json:"severity" yaml:"severity"`
	// Service is the affected service.
	Service string `json:"service" yaml:"service"`
	// Method is the affected method.
	Method string `json:"method,omitempty" yaml:"method,omitempty"`
	// Message describes the issue.
	Message string `json:"message" yaml:"message"`
	// Fix provides a fix suggestion.
	Fix string `json:"fix,omitempty" yaml:"fix,omitempty"`
}

// IssueSeverity represents the severity of an audit issue.
type IssueSeverity int

const (
	// SeverityInfo for informational issues.
	SeverityInfo IssueSeverity = iota
	// SeverityWarning for warning issues.
	SeverityWarning
	// SeverityError for error issues.
	SeverityError
)

// GeneratorOptions contains options for code generation.
type GeneratorOptions struct {
	// Framework is the authorization framework (go, casbin, opa).
	Framework string `json:"framework" yaml:"framework"`
	// Package is the generated package name.
	Package string `json:"package" yaml:"package"`
	// OutputDir is the output directory.
	OutputDir string `json:"output_dir" yaml:"output_dir"`
	// GenerateConstants generates constants for actions/roles/scopes.
	GenerateConstants bool `json:"generate_constants" yaml:"generate_constants"`
	// IncludeTests generates test files.
	IncludeTests bool `json:"include_tests" yaml:"include_tests"`
}
