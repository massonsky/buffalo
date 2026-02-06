package validation

import (
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════
//  ParseFieldAnnotation tests
// ══════════════════════════════════════════════════════════════════

func TestParseFieldAnnotation_DoubleRange(t *testing.T) {
	annotation := `[(buffalo.validate.rules).double = {gte: -90, lte: 90}]`

	rules, err := ParseFieldAnnotation(annotation, "lat", "double")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	assertRuleExists(t, rules, RuleGte, float64(-90))
	assertRuleExists(t, rules, RuleLte, float64(90))
}

func TestParseFieldAnnotation_DoubleRangeNegative(t *testing.T) {
	annotation := `[(buffalo.validate.rules).double = {gte: -180, lte: 180}]`

	rules, err := ParseFieldAnnotation(annotation, "lng", "double")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	assertRuleExists(t, rules, RuleGte, float64(-180))
	assertRuleExists(t, rules, RuleLte, float64(180))
}

func TestParseFieldAnnotation_Int32Rules(t *testing.T) {
	annotation := `[(buffalo.validate.rules).int32 = {gt: 0, lte: 150}]`

	rules, err := ParseFieldAnnotation(annotation, "age", "int32")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	assertRuleExists(t, rules, RuleGt, float64(0))
	assertRuleExists(t, rules, RuleLte, float64(150))
}

func TestParseFieldAnnotation_StringEmailWithLen(t *testing.T) {
	annotation := `[(buffalo.validate.rules).string = {min_len: 5, max_len: 255, email: true}]`

	rules, err := ParseFieldAnnotation(annotation, "email", "string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}

	assertRuleTypePresent(t, rules, RuleMinLen)
	assertRuleTypePresent(t, rules, RuleMaxLen)
	assertRuleTypePresent(t, rules, RuleEmail)
}

func TestParseFieldAnnotation_StringPattern(t *testing.T) {
	annotation := `[(buffalo.validate.rules).string = {pattern: "^[a-z]+$"}]`

	rules, err := ParseFieldAnnotation(annotation, "slug", "string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	if rules[0].Type != RulePattern {
		t.Errorf("expected rule type 'pattern', got '%s'", rules[0].Type)
	}
	if rules[0].Value != "^[a-z]+$" {
		t.Errorf("expected pattern '^[a-z]+$', got '%v'", rules[0].Value)
	}
}

func TestParseFieldAnnotation_Required(t *testing.T) {
	annotation := `[(buffalo.validate.rules).required = true]`

	rules, err := ParseFieldAnnotation(annotation, "name", "string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Type != RuleRequired {
		t.Errorf("expected 'required', got '%s'", rules[0].Type)
	}
}

func TestParseFieldAnnotation_RequiredFalse(t *testing.T) {
	annotation := `[(buffalo.validate.rules).required = false]`

	rules, err := ParseFieldAnnotation(annotation, "name", "string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules for required=false, got %d", len(rules))
	}
}

func TestParseFieldAnnotation_NoAnnotation(t *testing.T) {
	annotation := `[json_name = "latitude"]`

	rules, err := ParseFieldAnnotation(annotation, "lat", "double")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestParseFieldAnnotation_EmptyString(t *testing.T) {
	rules, err := ParseFieldAnnotation("", "x", "string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules for empty annotation, got %d", len(rules))
	}
}

func TestParseFieldAnnotation_InvalidNumericValue(t *testing.T) {
	annotation := `[(buffalo.validate.rules).double = {gte: not_a_number}]`

	_, err := ParseFieldAnnotation(annotation, "lat", "double")
	if err == nil {
		t.Error("expected error for non-numeric gte value, got nil")
	}
}

func TestParseFieldAnnotation_InvalidRegex(t *testing.T) {
	annotation := `[(buffalo.validate.rules).string = {pattern: "[invalid"}]`

	_, err := ParseFieldAnnotation(annotation, "x", "string")
	if err == nil {
		t.Error("expected error for invalid regex, got nil")
	}
}

func TestParseFieldAnnotation_UnknownRule(t *testing.T) {
	annotation := `[(buffalo.validate.rules).string = {bogus: 42}]`

	_, err := ParseFieldAnnotation(annotation, "x", "string")
	if err == nil {
		t.Error("expected error for unknown rule, got nil")
	}
}

func TestParseFieldAnnotation_RepeatedRules(t *testing.T) {
	annotation := `[(buffalo.validate.rules).repeated = {min_items: 1, max_items: 100, unique: true}]`

	rules, err := ParseFieldAnnotation(annotation, "tags", "repeated string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}

	assertRuleTypePresent(t, rules, RuleMinItems)
	assertRuleTypePresent(t, rules, RuleMaxItems)
	assertRuleTypePresent(t, rules, RuleUnique)
}

func TestParseFieldAnnotation_UUIDRule(t *testing.T) {
	annotation := `[(buffalo.validate.rules).string = {uuid: true}]`

	rules, err := ParseFieldAnnotation(annotation, "id", "string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Type != RuleUUID {
		t.Errorf("expected 'uuid', got '%s'", rules[0].Type)
	}
}

func TestParseFieldAnnotation_ConstRule(t *testing.T) {
	annotation := `[(buffalo.validate.rules).int32 = {const: 42}]`

	rules, err := ParseFieldAnnotation(annotation, "magic", "int32")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Type != RuleConst {
		t.Errorf("expected 'const', got '%s'", rules[0].Type)
	}
	if rules[0].Value != float64(42) {
		t.Errorf("expected const value 42, got %v", rules[0].Value)
	}
}

func TestParseFieldAnnotation_WithoutDotsShortForm(t *testing.T) {
	// Also support buffalo.validate (without .rules)
	annotation := `[(buffalo.validate).double = {gte: 0, lte: 100}]`

	rules, err := ParseFieldAnnotation(annotation, "score", "double")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
}

func TestParseFieldAnnotation_PrefixSuffixContains(t *testing.T) {
	annotation := `[(buffalo.validate.rules).string = {prefix: "http", suffix: ".com", contains: "://"}]`

	rules, err := ParseFieldAnnotation(annotation, "url", "string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}

	assertRuleTypePresent(t, rules, RulePrefix)
	assertRuleTypePresent(t, rules, RuleSuffix)
	assertRuleTypePresent(t, rules, RuleContains)
}

// ══════════════════════════════════════════════════════════════════
//  ExtractValidationRules tests
// ══════════════════════════════════════════════════════════════════

func TestExtractValidationRules_LocationMessage(t *testing.T) {
	proto := `syntax = "proto3";

package geo;

import "buffalo/validate/validate.proto";

message Location {
  double lat = 1 [(buffalo.validate.rules).double = {gte: -90, lte: 90}];
  double lng = 2 [(buffalo.validate.rules).double = {gte: -180, lte: 180}];
}
`
	results, err := ExtractValidationRules(proto, "location.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 message, got %d", len(results))
	}

	msg := results[0]
	if msg.MessageName != "Location" {
		t.Errorf("expected message name 'Location', got '%s'", msg.MessageName)
	}
	if msg.Package != "geo" {
		t.Errorf("expected package 'geo', got '%s'", msg.Package)
	}
	if len(msg.Fields) != 2 {
		t.Errorf("expected 2 validated fields, got %d", len(msg.Fields))
	}
	if _, ok := msg.Fields["lat"]; !ok {
		t.Error("expected 'lat' field in rules")
	}
	if _, ok := msg.Fields["lng"]; !ok {
		t.Error("expected 'lng' field in rules")
	}
}

func TestExtractValidationRules_MultipleMessages(t *testing.T) {
	proto := `syntax = "proto3";

package api;

message User {
  string email = 1 [(buffalo.validate.rules).string = {email: true, min_len: 5}];
  string name  = 2 [(buffalo.validate.rules).string = {min_len: 1, max_len: 128}];
  int32 age    = 3 [(buffalo.validate.rules).int32 = {gt: 0, lte: 150}];
}

message Location {
  double lat = 1 [(buffalo.validate.rules).double = {gte: -90, lte: 90}];
  double lng = 2 [(buffalo.validate.rules).double = {gte: -180, lte: 180}];
}

message Tag {
  string label = 1;
}
`
	results, err := ExtractValidationRules(proto, "test.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Tag has no validation annotations, so only 2 messages
	if len(results) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(results))
	}

	names := map[string]bool{}
	for _, r := range results {
		names[r.MessageName] = true
	}
	if !names["User"] {
		t.Error("expected User message")
	}
	if !names["Location"] {
		t.Error("expected Location message")
	}
}

func TestExtractValidationRules_NoAnnotations(t *testing.T) {
	proto := `syntax = "proto3";

package test;

message Simple {
  string name = 1;
  int32 value = 2;
}
`
	results, err := ExtractValidationRules(proto, "simple.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 messages with rules, got %d", len(results))
	}
}

func TestExtractValidationRules_MixedAnnotatedAndPlain(t *testing.T) {
	proto := `syntax = "proto3";

package test;

message Config {
  string name    = 1 [(buffalo.validate.rules).string = {not_empty: true}];
  string comment = 2;
  int32 priority = 3 [(buffalo.validate.rules).int32 = {gte: 0, lte: 100}];
}
`
	results, err := ExtractValidationRules(proto, "config.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 message, got %d", len(results))
	}
	if len(results[0].Fields) != 2 {
		t.Errorf("expected 2 validated fields, got %d", len(results[0].Fields))
	}
}

func TestExtractValidationRules_RepeatedField(t *testing.T) {
	proto := `syntax = "proto3";

package test;

message Route {
  string id = 1 [(buffalo.validate.rules).required = true];
  repeated string waypoints = 2 [(buffalo.validate.rules).repeated = {min_items: 2, max_items: 100}];
}
`
	results, err := ExtractValidationRules(proto, "route.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 message, got %d", len(results))
	}
	if len(results[0].Fields) != 2 {
		t.Errorf("expected 2 validated fields, got %d", len(results[0].Fields))
	}
}

func TestExtractValidationRules_PackageExtraction(t *testing.T) {
	proto := `syntax = "proto3";

package my.company.api.v1;

message Foo {
  string bar = 1 [(buffalo.validate.rules).string = {not_empty: true}];
}
`
	results, err := ExtractValidationRules(proto, "foo.proto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 message, got %d", len(results))
	}
	if results[0].Package != "my.company.api.v1" {
		t.Errorf("expected package 'my.company.api.v1', got '%s'", results[0].Package)
	}
}

// ══════════════════════════════════════════════════════════════════
//  parseFieldLine tests
// ══════════════════════════════════════════════════════════════════

func TestParseFieldLine_Simple(t *testing.T) {
	f := parseFieldLine(`  string name = 1;`)
	if f == nil {
		t.Fatal("expected non-nil field")
	}
	if f.name != "name" {
		t.Errorf("expected field name 'name', got '%s'", f.name)
	}
	if f.typ != "string" {
		t.Errorf("expected type 'string', got '%s'", f.typ)
	}
}

func TestParseFieldLine_WithOptions(t *testing.T) {
	f := parseFieldLine(`  double lat = 1 [(buffalo.validate.rules).double = {gte: -90, lte: 90}];`)
	if f == nil {
		t.Fatal("expected non-nil field")
	}
	if f.name != "lat" {
		t.Errorf("expected field name 'lat', got '%s'", f.name)
	}
	if f.options == "" {
		t.Error("expected non-empty options")
	}
	if !strings.Contains(f.options, "buffalo.validate") {
		t.Errorf("expected options to contain 'buffalo.validate', got '%s'", f.options)
	}
}

func TestParseFieldLine_Repeated(t *testing.T) {
	f := parseFieldLine(`  repeated string tags = 5 [(buffalo.validate.rules).repeated = {min_items: 1}];`)
	if f == nil {
		t.Fatal("expected non-nil field")
	}
	if f.name != "tags" {
		t.Errorf("expected field name 'tags', got '%s'", f.name)
	}
	if !strings.HasPrefix(f.typ, "repeated") {
		t.Errorf("expected type starting with 'repeated', got '%s'", f.typ)
	}
}

func TestParseFieldLine_CommentOnly(t *testing.T) {
	f := parseFieldLine(`  // This is a comment`)
	if f != nil {
		t.Error("expected nil for comment-only line")
	}
}

func TestParseFieldLine_Empty(t *testing.T) {
	f := parseFieldLine(`  }`)
	if f != nil {
		t.Error("expected nil for closing brace")
	}
}

func TestParseFieldLine_WithTrailingComment(t *testing.T) {
	f := parseFieldLine(`  double lat = 1 [(buffalo.validate.rules).double = {gte: -90}]; // latitude`)
	if f == nil {
		t.Fatal("expected non-nil field")
	}
	if f.name != "lat" {
		t.Errorf("expected field name 'lat', got '%s'", f.name)
	}
	if !strings.Contains(f.options, "buffalo.validate") {
		t.Errorf("expected options to contain buffalo.validate, got '%s'", f.options)
	}
}

// ══════════════════════════════════════════════════════════════════
//  Helpers
// ══════════════════════════════════════════════════════════════════

func assertRuleExists(t *testing.T, rules []FieldRule, ruleType RuleType, expectedValue float64) {
	t.Helper()
	for _, r := range rules {
		if r.Type == ruleType {
			if v, ok := r.Value.(float64); ok && v == expectedValue {
				return
			}
			t.Errorf("rule '%s' has value %v, expected %v", ruleType, r.Value, expectedValue)
			return
		}
	}
	t.Errorf("rule '%s' not found in rules", ruleType)
}

func assertRuleTypePresent(t *testing.T, rules []FieldRule, ruleType RuleType) {
	t.Helper()
	for _, r := range rules {
		if r.Type == ruleType {
			return
		}
	}
	t.Errorf("rule type '%s' not found in rules", ruleType)
}
