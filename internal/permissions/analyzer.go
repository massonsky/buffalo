package permissions

import (
	"context"
	"fmt"
	"strings"
)

// Analyzer analyzes permissions for issues and generates reports.
type Analyzer struct {
	rules []AuditRule
}

// AuditRule defines a rule for auditing permissions.
type AuditRule struct {
	ID          string
	Name        string
	Description string
	Severity    IssueSeverity
	Check       func(service *ServicePermissions, method string, perm *Permission) *AuditIssue
}

// NewAnalyzer creates a new permission analyzer with default rules.
func NewAnalyzer() *Analyzer {
	a := &Analyzer{}
	a.rules = a.defaultRules()
	return a
}

// AddRule adds a custom audit rule.
func (a *Analyzer) AddRule(rule AuditRule) {
	a.rules = append(a.rules, rule)
}

// Audit analyzes permissions and returns audit issues.
func (a *Analyzer) Audit(ctx context.Context, services []*ServicePermissions) []AuditIssue {
	var issues []AuditIssue

	for _, svc := range services {
		// Check service-level issues
		svcIssues := a.auditService(svc)
		issues = append(issues, svcIssues...)

		// Check method-level issues
		for method, perm := range svc.Methods {
			for _, rule := range a.rules {
				if issue := rule.Check(svc, method, perm); issue != nil {
					issue.Service = svc.Service
					issue.Method = method
					issues = append(issues, *issue)
				}
			}
		}
	}

	return issues
}

func (a *Analyzer) auditService(svc *ServicePermissions) []AuditIssue {
	var issues []AuditIssue

	// Check if resource is set
	if svc.Resource == "" {
		issues = append(issues, AuditIssue{
			Service:  svc.Service,
			RuleID:   "MISSING_RESOURCE",
			Severity: SeverityWarning,
			Message:  "Service has no resource defined",
			Fix:      "Add option (buffalo.permissions.resource) = \"resource_name\"",
		})
	}

	return issues
}

func (a *Analyzer) defaultRules() []AuditRule {
	return []AuditRule{
		{
			ID:          "NO_ROLES_OR_SCOPES",
			Name:        "Missing Authorization",
			Description: "Method has no roles or scopes defined and is not marked as public",
			Severity:    SeverityError,
			Check: func(svc *ServicePermissions, method string, perm *Permission) *AuditIssue {
				if perm.Public {
					return nil
				}
				if len(perm.Roles) == 0 && len(perm.Scopes) == 0 {
					return &AuditIssue{
						RuleID:   "NO_ROLES_OR_SCOPES",
						Severity: SeverityError,
						Message:  "Method has no roles or scopes defined",
						Fix:      "Add roles, scopes, or mark as public: true",
					}
				}
				return nil
			},
		},
		{
			ID:          "WILDCARD_ROLE",
			Name:        "Wildcard Role",
			Description: "Method allows all roles with wildcard",
			Severity:    SeverityWarning,
			Check: func(svc *ServicePermissions, method string, perm *Permission) *AuditIssue {
				for _, role := range perm.Roles {
					if role == "*" || role == "any" {
						return &AuditIssue{
							RuleID:   "WILDCARD_ROLE",
							Severity: SeverityWarning,
							Message:  "Method uses wildcard role which allows any authenticated user",
							Fix:      "Specify explicit roles instead of wildcard",
						}
					}
				}
				return nil
			},
		},
		{
			ID:          "ADMIN_NO_MFA",
			Name:        "Admin Without MFA",
			Description: "Admin role does not require MFA",
			Severity:    SeverityWarning,
			Check: func(svc *ServicePermissions, method string, perm *Permission) *AuditIssue {
				hasAdmin := false
				for _, role := range perm.Roles {
					if strings.Contains(strings.ToLower(role), "admin") ||
						strings.Contains(strings.ToLower(role), "superuser") {
						hasAdmin = true
						break
					}
				}
				if hasAdmin && !perm.RequireMFA {
					return &AuditIssue{
						RuleID:   "ADMIN_NO_MFA",
						Severity: SeverityWarning,
						Message:  "Admin access does not require MFA",
						Fix:      "Add require_mfa: true for admin operations",
					}
				}
				return nil
			},
		},
		{
			ID:          "SENSITIVE_NO_AUDIT",
			Name:        "Sensitive Operation Without Audit",
			Description: "Sensitive operation does not have audit logging enabled",
			Severity:    SeverityWarning,
			Check: func(svc *ServicePermissions, method string, perm *Permission) *AuditIssue {
				// Check if it's a sensitive operation (write, delete, admin)
				sensitiveActions := []string{"write", "delete", "admin", "create", "update", "modify"}
				isSensitive := false
				for _, sa := range sensitiveActions {
					if strings.Contains(strings.ToLower(perm.Action), sa) {
						isSensitive = true
						break
					}
				}

				// Also check method name
				sensitiveMethodPatterns := []string{"Delete", "Remove", "Create", "Update", "Modify", "Admin"}
				for _, pattern := range sensitiveMethodPatterns {
					if strings.Contains(method, pattern) {
						isSensitive = true
						break
					}
				}

				if isSensitive && !perm.AuditLog {
					return &AuditIssue{
						RuleID:   "SENSITIVE_NO_AUDIT",
						Severity: SeverityWarning,
						Message:  "Sensitive operation does not have audit logging enabled",
						Fix:      "Add audit_log: true for sensitive operations",
					}
				}
				return nil
			},
		},
		{
			ID:          "PUBLIC_WRITE",
			Name:        "Public Write Operation",
			Description: "Write operation is marked as public",
			Severity:    SeverityError,
			Check: func(svc *ServicePermissions, method string, perm *Permission) *AuditIssue {
				if !perm.Public {
					return nil
				}
				writeActions := []string{"write", "delete", "create", "update", "modify", "admin"}
				for _, wa := range writeActions {
					if strings.Contains(strings.ToLower(perm.Action), wa) {
						return &AuditIssue{
							RuleID:   "PUBLIC_WRITE",
							Severity: SeverityError,
							Message:  "Write operation is marked as public",
							Fix:      "Remove public: true and add appropriate roles",
						}
					}
				}
				return nil
			},
		},
		{
			ID:          "MISSING_RATE_LIMIT",
			Name:        "Missing Rate Limit",
			Description: "Public endpoint has no rate limit",
			Severity:    SeverityInfo,
			Check: func(svc *ServicePermissions, method string, perm *Permission) *AuditIssue {
				if perm.Public && perm.RateLimit == nil {
					return &AuditIssue{
						RuleID:   "MISSING_RATE_LIMIT",
						Severity: SeverityInfo,
						Message:  "Public endpoint has no rate limit configured",
						Fix:      "Add rate_limit: { requests: 100, window: \"1m\" }",
					}
				}
				return nil
			},
		},
		{
			ID:          "INCONSISTENT_NAMING",
			Name:        "Inconsistent Action Naming",
			Description: "Action does not follow naming convention",
			Severity:    SeverityInfo,
			Check: func(svc *ServicePermissions, method string, perm *Permission) *AuditIssue {
				if perm.Action == "" {
					return &AuditIssue{
						RuleID:   "INCONSISTENT_NAMING",
						Severity: SeverityInfo,
						Message:  "Method has no action defined",
						Fix:      "Add action: \"resource:verb\" (e.g., \"users:read\")",
					}
				}

				// Check for resource:action format
				if !strings.Contains(perm.Action, ":") {
					return &AuditIssue{
						RuleID:   "INCONSISTENT_NAMING",
						Severity: SeverityInfo,
						Message:  fmt.Sprintf("Action '%s' does not follow resource:verb pattern", perm.Action),
						Fix:      "Use format: \"resource:verb\" (e.g., \"users:read\")",
					}
				}
				return nil
			},
		},
		{
			ID:          "SELF_ACCESS_NO_CONDITION",
			Name:        "Self Access Without Condition",
			Description: "allow_self is set but no owner condition defined",
			Severity:    SeverityWarning,
			Check: func(svc *ServicePermissions, method string, perm *Permission) *AuditIssue {
				if !perm.AllowSelf {
					return nil
				}

				hasOwnerCondition := false
				for _, cond := range perm.Conditions {
					if cond.Field == "owner_id" || cond.Field == "user_id" ||
						strings.Contains(cond.Source, "user") {
						hasOwnerCondition = true
						break
					}
				}

				if !hasOwnerCondition {
					return &AuditIssue{
						RuleID:   "SELF_ACCESS_NO_CONDITION",
						Severity: SeverityWarning,
						Message:  "allow_self is set but no owner condition defined",
						Fix:      "Add condition: { field: \"owner_id\", operator: \"eq\", source: \"auth.user_id\" }",
					}
				}
				return nil
			},
		},
	}
}

// Diff compares two permission sets and returns differences.
func (a *Analyzer) Diff(ctx context.Context, old, new []*ServicePermissions) []PermissionDiff {
	var diffs []PermissionDiff

	oldMap := make(map[string]map[string]*Permission)
	for _, svc := range old {
		oldMap[svc.Service] = svc.Methods
	}

	newMap := make(map[string]map[string]*Permission)
	for _, svc := range new {
		newMap[svc.Service] = svc.Methods
	}

	// Check for added/modified services
	for serviceName, newMethods := range newMap {
		oldMethods, exists := oldMap[serviceName]
		if !exists {
			// New service
			for method, perm := range newMethods {
				diffs = append(diffs, PermissionDiff{
					Type:    DiffAdded,
					Service: serviceName,
					Method:  method,
					New:     perm,
				})
			}
			continue
		}

		// Check methods
		for method, newPerm := range newMethods {
			oldPerm, exists := oldMethods[method]
			if !exists {
				diffs = append(diffs, PermissionDiff{
					Type:    DiffAdded,
					Service: serviceName,
					Method:  method,
					New:     newPerm,
				})
				continue
			}

			// Check for modifications
			if !permissionsEqual(oldPerm, newPerm) {
				diffs = append(diffs, PermissionDiff{
					Type:    DiffModified,
					Service: serviceName,
					Method:  method,
					Old:     oldPerm,
					New:     newPerm,
				})
			}
		}

		// Check for removed methods
		for method, oldPerm := range oldMethods {
			if _, exists := newMethods[method]; !exists {
				diffs = append(diffs, PermissionDiff{
					Type:    DiffRemoved,
					Service: serviceName,
					Method:  method,
					Old:     oldPerm,
				})
			}
		}
	}

	// Check for removed services
	for serviceName, oldMethods := range oldMap {
		if _, exists := newMap[serviceName]; !exists {
			for method, perm := range oldMethods {
				diffs = append(diffs, PermissionDiff{
					Type:    DiffRemoved,
					Service: serviceName,
					Method:  method,
					Old:     perm,
				})
			}
		}
	}

	return diffs
}

func permissionsEqual(a, b *Permission) bool {
	if a.Action != b.Action || a.Public != b.Public ||
		a.AllowSelf != b.AllowSelf || a.RequireMFA != b.RequireMFA ||
		a.AuditLog != b.AuditLog {
		return false
	}

	if !stringSliceEqual(a.Roles, b.Roles) || !stringSliceEqual(a.Scopes, b.Scopes) {
		return false
	}

	if len(a.Conditions) != len(b.Conditions) {
		return false
	}

	for i, cond := range a.Conditions {
		if cond != b.Conditions[i] {
			return false
		}
	}

	return true
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Summary returns a summary of permissions.
func (a *Analyzer) Summary(services []*ServicePermissions) PermissionSummary {
	summary := PermissionSummary{
		ByRole:  make(map[string]int),
		ByScope: make(map[string]int),
	}

	for _, svc := range services {
		summary.ServiceCount++
		for _, perm := range svc.Methods {
			summary.MethodCount++
			if perm.Public {
				summary.PublicCount++
			}

			for _, role := range perm.Roles {
				summary.ByRole[role]++
			}
			for _, scope := range perm.Scopes {
				summary.ByScope[scope]++
			}
		}
	}

	return summary
}

// PermissionSummary holds summary statistics.
type PermissionSummary struct {
	ServiceCount int
	MethodCount  int
	PublicCount  int
	ByRole       map[string]int
	ByScope      map[string]int
}
