package bazel

import (
	"bytes"
	"context"
	"encoding/xml"
	"os/exec"
	"strings"
)

// BazelQuerier runs `bazel query` / `bazel cquery` to inspect the build graph.
type BazelQuerier struct {
	workspaceRoot string
	bazelPath     string
}

// NewQuerier creates a querier rooted at the given workspace.
func NewQuerier(workspaceRoot string) *BazelQuerier {
	return &BazelQuerier{
		workspaceRoot: workspaceRoot,
		bazelPath:     "bazel",
	}
}

// SetBazelPath overrides the bazel binary path.
func (q *BazelQuerier) SetBazelPath(path string) {
	q.bazelPath = path
}

// FindProtoTargets uses `bazel query` to find all proto_library targets.
func (q *BazelQuerier) FindProtoTargets(ctx context.Context, pattern string) ([]string, error) {
	if pattern == "" {
		pattern = "//..."
	}
	query := "kind(proto_library, " + pattern + ")"
	return q.runQuery(ctx, query)
}

// GetDeps returns the direct dependencies of a target.
func (q *BazelQuerier) GetDeps(ctx context.Context, target string) ([]string, error) {
	query := "deps(" + target + ", 1)"
	return q.runQuery(ctx, query)
}

// GetTransitiveDeps returns all transitive deps of a target.
func (q *BazelQuerier) GetTransitiveDeps(ctx context.Context, target string) ([]string, error) {
	query := "deps(" + target + ")"
	return q.runQuery(ctx, query)
}

// GetProtoSources returns the proto source files for a target using --output=xml.
func (q *BazelQuerier) GetProtoSources(ctx context.Context, target string) ([]string, error) {
	args := []string{"query", "--output=xml", target}
	out, err := q.run(ctx, args)
	if err != nil {
		return nil, err
	}
	return parseProtoSourcesFromXML(out), nil
}

// BuildQueryResult runs bazel query for a set of targets and returns
// a structured QueryResult with deps and source files.
func (q *BazelQuerier) BuildQueryResult(ctx context.Context, targets []string) (*QueryResult, error) {
	result := &QueryResult{
		Targets:    targets,
		Deps:       make(map[string][]string),
		ProtoFiles: make(map[string][]string),
	}

	for _, t := range targets {
		deps, err := q.GetDeps(ctx, t)
		if err != nil {
			// Non-fatal: we may not have bazel available
			continue
		}
		result.Deps[t] = deps

		sources, err := q.GetProtoSources(ctx, t)
		if err != nil {
			continue
		}
		result.ProtoFiles[t] = sources
	}

	return result, nil
}

// runQuery runs `bazel query` and returns the output lines.
func (q *BazelQuerier) runQuery(ctx context.Context, query string) ([]string, error) {
	args := []string{"query", query, "--keep_going", "--noshow_progress"}
	out, err := q.run(ctx, args)
	if err != nil {
		return nil, err
	}

	var targets []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "WARNING") && !strings.HasPrefix(line, "INFO") && !strings.HasPrefix(line, "Loading") {
			targets = append(targets, line)
		}
	}
	return targets, nil
}

// run executes a bazel command and returns stdout.
func (q *BazelQuerier) run(ctx context.Context, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, q.bazelPath, args...)
	cmd.Dir = q.workspaceRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", &BazelError{
			Command: q.bazelPath + " " + strings.Join(args, " "),
			Stderr:  stderr.String(),
			Err:     err,
		}
	}

	return stdout.String(), nil
}

// BazelError wraps errors from bazel commands.
type BazelError struct {
	Command string
	Stderr  string
	Err     error
}

func (e *BazelError) Error() string {
	msg := "bazel command failed: " + e.Command
	if e.Stderr != "" {
		// Trim long stderr
		stderr := e.Stderr
		if len(stderr) > 500 {
			stderr = stderr[:500] + "..."
		}
		msg += "\n" + stderr
	}
	return msg
}

func (e *BazelError) Unwrap() error {
	return e.Err
}

// IsBazelAvailable checks if the bazel binary is available in PATH.
func IsBazelAvailable() bool {
	_, err := exec.LookPath("bazel")
	return err == nil
}

// xmlQueryResult is used to parse --output=xml from bazel query.
type xmlQueryResult struct {
	XMLName xml.Name  `xml:"query"`
	Rules   []xmlRule `xml:"rule"`
}

type xmlRule struct {
	Class   string          `xml:"class,attr"`
	Name    string          `xml:"name,attr"`
	Lists   []xmlListAttr   `xml:"list"`
	Strings []xmlStringAttr `xml:"string"`
}

type xmlListAttr struct {
	Name   string          `xml:"name,attr"`
	Labels []xmlLabelEntry `xml:"label"`
}

type xmlStringAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type xmlLabelEntry struct {
	Value string `xml:"value,attr"`
}

func parseProtoSourcesFromXML(xmlContent string) []string {
	var result xmlQueryResult
	if err := xml.Unmarshal([]byte(xmlContent), &result); err != nil {
		return nil
	}

	var sources []string
	for _, rule := range result.Rules {
		for _, list := range rule.Lists {
			if list.Name == "srcs" {
				for _, label := range list.Labels {
					sources = append(sources, label.Value)
				}
			}
		}
	}
	return sources
}
