package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// structToMapstructureMap converts any struct annotated with `mapstructure` tags
// into a generic map suitable for YAML/JSON marshaling. Honors:
//   - tag "-" -> skip
//   - tag option ",omitempty" -> drop zero values
//   - pointers -> deref or drop if nil
func structToMapstructureMap(v any) (any, error) {
	return walkValue(reflect.ValueOf(v))
}

func walkValue(v reflect.Value) (any, error) {
	if !v.IsValid() {
		return nil, nil
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil, nil
		}
		return walkValue(v.Elem())
	case reflect.Struct:
		return walkStruct(v)
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return nil, nil
		}
		out := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem, err := walkValue(v.Index(i))
			if err != nil {
				return nil, err
			}
			out = append(out, elem)
		}
		return out, nil
	case reflect.Map:
		if v.IsNil() {
			return nil, nil
		}
		out := map[string]any{}
		iter := v.MapRange()
		for iter.Next() {
			key := fmt.Sprintf("%v", iter.Key().Interface())
			val, err := walkValue(iter.Value())
			if err != nil {
				return nil, err
			}
			out[key] = val
		}
		return out, nil
	default:
		return v.Interface(), nil
	}
}

func walkStruct(v reflect.Value) (map[string]any, error) {
	t := v.Type()
	out := map[string]any{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("mapstructure")
		if tag == "-" {
			continue
		}
		name, opts := parseTag(tag)
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		fv := v.Field(i)
		if hasOpt(opts, "omitempty") && fv.IsZero() {
			continue
		}
		val, err := walkValue(fv)
		if err != nil {
			return nil, err
		}
		out[name] = val
	}
	return out, nil
}

func parseTag(tag string) (string, []string) {
	if tag == "" {
		return "", nil
	}
	parts := strings.Split(tag, ",")
	return parts[0], parts[1:]
}

func hasOpt(opts []string, want string) bool {
	for _, o := range opts {
		if o == want {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// JSON Schema generation (Draft 2020-12 subset, fully self-contained, no deps).
// ---------------------------------------------------------------------------

// GenerateJSONSchema produces a JSON Schema document describing buffalo.yaml.
// Source of truth: the Config struct and its `mapstructure` / `desc` tags.
func GenerateJSONSchema() ([]byte, error) {
	defs := map[string]any{}
	root := schemaForType(reflect.TypeOf(Config{}), defs)

	doc := map[string]any{
		"$schema":     "https://json-schema.org/draft/2020-12/schema",
		"$id":         "https://github.com/massonsky/buffalo/schemas/buffalo.schema.json",
		"title":       "Buffalo Configuration",
		"description": fmt.Sprintf("Schema for buffalo.yaml. Schema version: %d.", CurrentSchemaVersion),
		"type":        "object",
	}
	for k, v := range root {
		doc[k] = v
	}
	if len(defs) > 0 {
		doc["$defs"] = defs
	}
	return json.MarshalIndent(doc, "", "  ")
}

func schemaForType(t reflect.Type, defs map[string]any) map[string]any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": schemaForType(t.Elem(), defs),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": schemaForType(t.Elem(), defs),
		}
	case reflect.Interface:
		return map[string]any{}
	case reflect.Struct:
		return schemaForStruct(t, defs)
	}
	return map[string]any{}
}

func schemaForStruct(t reflect.Type, defs map[string]any) map[string]any {
	// Use $ref for non-root structs to keep document small and cycle-safe.
	defName := t.Name()
	if defName != "" && t != reflect.TypeOf(Config{}) {
		if _, ok := defs[defName]; !ok {
			defs[defName] = map[string]any{"$comment": "placeholder"} // reserve slot
			defs[defName] = buildStructSchema(t, defs)
		}
		return map[string]any{"$ref": "#/$defs/" + defName}
	}
	return buildStructSchema(t, defs)
}

func buildStructSchema(t reflect.Type, defs map[string]any) map[string]any {
	props := map[string]any{}
	keys := []string{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("mapstructure")
		if tag == "-" {
			continue
		}
		name, _ := parseTag(tag)
		if name == "" {
			name = strings.ToLower(f.Name)
		}
		s := schemaForType(f.Type, defs)
		if desc := f.Tag.Get("desc"); desc != "" {
			s["description"] = desc
		}
		props[name] = s
		keys = append(keys, name)
	}
	sort.Strings(keys)
	ordered := map[string]any{}
	for _, k := range keys {
		ordered[k] = props[k]
	}
	return map[string]any{
		"type":                 "object",
		"properties":           ordered,
		"additionalProperties": false,
	}
}
