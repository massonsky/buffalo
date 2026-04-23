package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SchemaMigration mutates a buffalo.yaml YAML node in-place to upgrade it
// from one schema version to the next. Migrations are pure: they only touch
// `node`, never the filesystem.
type SchemaMigration struct {
	From        int
	To          int
	Description string
	Apply       func(root *yaml.Node) error
}

// schemaMigrations is the registered chain of v1 -> v2 -> ... migrations.
// Each entry must increment the version by exactly one.
var schemaMigrations = []SchemaMigration{
	{
		From:        1,
		To:          2,
		Description: "Stamp schema_version: 2 (no shape changes from v1).",
		Apply: func(root *yaml.Node) error {
			return setMappingScalar(root, "schema_version", "2", "!!int")
		},
	},
}

// MigrateYAMLBytes applies all schema migrations needed to bring `data` from
// its current version up to CurrentSchemaVersion. Returns the migrated bytes,
// the source version, and the target version. Source version is inferred:
// missing `schema_version` -> 1.
func MigrateYAMLBytes(data []byte) (out []byte, from, to int, err error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, 0, 0, fmt.Errorf("parse YAML: %w", err)
	}

	root := documentRoot(&doc)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, 0, 0, fmt.Errorf("config root must be a YAML mapping")
	}

	from = readSchemaVersion(root)
	to = CurrentSchemaVersion
	if from == to {
		return data, from, to, nil
	}
	if from > to {
		return nil, from, to, fmt.Errorf(
			"schema_version %d is newer than supported (%d); upgrade buffalo", from, to)
	}

	for v := from; v < to; v++ {
		mig := findMigration(v)
		if mig == nil {
			return nil, from, to, fmt.Errorf(
				"no migration registered for schema_version %d -> %d", v, v+1)
		}
		if err := mig.Apply(root); err != nil {
			return nil, from, to, fmt.Errorf("migrate %d->%d: %w", v, v+1, err)
		}
	}

	out, err = marshalDoc(&doc)
	if err != nil {
		return nil, from, to, err
	}
	return out, from, to, nil
}

// MigrateFile reads `path`, applies migrations, and writes the result back.
// When dryRun is true the file is not modified; the new contents are returned.
func MigrateFile(path string, dryRun bool) (out []byte, from, to int, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, err
	}
	out, from, to, err = MigrateYAMLBytes(data)
	if err != nil {
		return nil, from, to, err
	}
	if dryRun || from == to {
		return out, from, to, nil
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return nil, from, to, err
	}
	return out, from, to, nil
}

// ---------- helpers ----------

func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

func readSchemaVersion(root *yaml.Node) int {
	for i := 0; i+1 < len(root.Content); i += 2 {
		k := root.Content[i]
		v := root.Content[i+1]
		if k.Value == "schema_version" && v.Kind == yaml.ScalarNode {
			var n int
			if _, err := fmt.Sscanf(v.Value, "%d", &n); err == nil {
				return n
			}
		}
	}
	return 1
}

func setMappingScalar(root *yaml.Node, key, value, tag string) error {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			root.Content[i+1].Value = value
			root.Content[i+1].Tag = tag
			root.Content[i+1].Kind = yaml.ScalarNode
			return nil
		}
	}
	root.Content = append([]*yaml.Node{
		{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"},
		{Kind: yaml.ScalarNode, Value: value, Tag: tag},
	}, root.Content...)
	return nil
}

func findMigration(from int) *SchemaMigration {
	for i := range schemaMigrations {
		if schemaMigrations[i].From == from {
			return &schemaMigrations[i]
		}
	}
	return nil
}

func marshalDoc(doc *yaml.Node) ([]byte, error) {
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal YAML: %w", err)
	}
	return out, nil
}
