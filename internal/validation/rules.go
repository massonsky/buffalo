// Package validation provides proto field validation rule parsing and code generation.
//
// It implements a system inspired by protoc-gen-validate (PGV) as a native
// Buffalo feature. Users annotate proto fields with [(buffalo.validate.rules)...]
// options, and Buffalo parses these annotations to generate Validate() methods
// for each target language.
package validation

import (
	"fmt"
	"strings"
)

// RuleType identifies a specific kind of validation rule.
type RuleType string

const (
	RuleRequired RuleType = "required"
	RuleGt       RuleType = "gt"
	RuleGte      RuleType = "gte"
	RuleLt       RuleType = "lt"
	RuleLte      RuleType = "lte"
	RuleIn       RuleType = "in"
	RuleNotIn    RuleType = "not_in"
	RuleConst    RuleType = "const"
	RuleMinLen   RuleType = "min_len"
	RuleMaxLen   RuleType = "max_len"
	RulePattern  RuleType = "pattern"
	RuleEmail    RuleType = "email"
	RuleURI      RuleType = "uri"
	RuleUUID     RuleType = "uuid"
	RuleNotEmpty RuleType = "not_empty"
	RulePrefix   RuleType = "prefix"
	RuleSuffix   RuleType = "suffix"
	RuleContains RuleType = "contains"
	RuleIP       RuleType = "ip"
	RuleIPv4     RuleType = "ipv4"
	RuleIPv6     RuleType = "ipv6"
	RuleHostname RuleType = "hostname"
	RuleMinItems RuleType = "min_items"
	RuleMaxItems RuleType = "max_items"
	RuleUnique   RuleType = "unique"
	RuleMinPairs RuleType = "min_pairs"
	RuleMaxPairs RuleType = "max_pairs"
	RuleGtNow    RuleType = "gt_now"
	RuleLtNow    RuleType = "lt_now"
)

// FieldRule represents a single parsed validation rule attached to a field.
type FieldRule struct {
	Type      RuleType    // Kind of rule
	Value     interface{} // Rule parameter (threshold, pattern, list, etc.)
	FieldName string      // Proto field name this applies to
	FieldType string      // Proto field type (string, int32, double, etc.)
	Message   string      // Custom error message override
}

// MessageRules is a collection of validation rules for a single message.
type MessageRules struct {
	MessageName string                 // Proto message name
	Package     string                 // Proto package
	FilePath    string                 // Source .proto file path
	Disabled    bool                   // If true, skip validation for this message
	Fields      map[string][]FieldRule // field_name -> list of rules
}

// Violation describes a single validation failure.
type Violation struct {
	Field   string   // Field that failed validation
	Rule    RuleType // Rule that was violated
	Message string   // Human-readable error message
}

// Error implements the error interface for Violation.
func (v Violation) Error() string {
	return fmt.Sprintf("field '%s' failed rule '%s': %s", v.Field, v.Rule, v.Message)
}

// ValidationResult contains all violations for a validated message instance.
type ValidationResult struct {
	MessageName string
	Violations  []Violation
}

// IsValid returns true if no violations were found.
func (r *ValidationResult) IsValid() bool {
	return len(r.Violations) == 0
}

// Error returns a combined error string or empty string if valid.
func (r *ValidationResult) Error() string {
	if r.IsValid() {
		return ""
	}
	msgs := make([]string, len(r.Violations))
	for i, v := range r.Violations {
		msgs[i] = v.Error()
	}
	return fmt.Sprintf("validation failed for %s: [%s]", r.MessageName, strings.Join(msgs, "; "))
}
