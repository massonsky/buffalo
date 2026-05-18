package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitKeepsExistingConfigAndContinues(t *testing.T) {
	runInTempDir(t)
	oldForce, oldBazel, oldExamples := initForce, initBazel, initExamples
	initForce, initBazel, initExamples = false, true, false
	t.Cleanup(func() {
		initForce, initBazel, initExamples = oldForce, oldBazel, oldExamples
	})

	existingConfig := "project:\n  name: existing\n"
	if err := os.WriteFile("buffalo.yaml", []byte(existingConfig), 0o600); err != nil {
		t.Fatalf("failed to seed buffalo.yaml: %v", err)
	}

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit should continue with an existing config: %v", err)
	}

	data, err := os.ReadFile("buffalo.yaml")
	if err != nil {
		t.Fatalf("failed to read buffalo.yaml: %v", err)
	}
	if string(data) != existingConfig {
		t.Fatalf("runInit rewrote existing config:\n%s", data)
	}
	for _, path := range []string{
		filepath.Join("generated"),
		filepath.Join(".buffalo", "proto", "buffalo", "validate", "validate.proto"),
		filepath.Join(".buffalo", "bazel", "rules_buffalo", "buffalo", "defs.bzl"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected init artifact %s: %v", path, err)
		}
	}
	if _, err := os.Stat("protos"); !os.IsNotExist(err) {
		t.Fatalf("init without --examples should not create protos directory, stat err: %v", err)
	}
}

func TestRunInitForceOverwritesExistingConfig(t *testing.T) {
	runInTempDir(t)
	oldForce, oldBazel, oldExamples := initForce, initBazel, initExamples
	initForce, initBazel, initExamples = true, false, false
	t.Cleanup(func() {
		initForce, initBazel, initExamples = oldForce, oldBazel, oldExamples
	})

	if err := os.WriteFile("buffalo.yaml", []byte("project:\n  name: existing\n"), 0o600); err != nil {
		t.Fatalf("failed to seed buffalo.yaml: %v", err)
	}

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	data, err := os.ReadFile("buffalo.yaml")
	if err != nil {
		t.Fatalf("failed to read buffalo.yaml: %v", err)
	}
	if !strings.Contains(string(data), "name: my-proto-project") {
		t.Fatalf("--force should rewrite config with defaults:\n%s", data)
	}
}

func TestRunInitExamplesCreatesProtoExample(t *testing.T) {
	runInTempDir(t)
	oldForce, oldBazel, oldExamples := initForce, initBazel, initExamples
	initForce, initBazel, initExamples = false, false, true
	t.Cleanup(func() {
		initForce, initBazel, initExamples = oldForce, oldBazel, oldExamples
	})

	if err := runInit(nil, nil); err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	examplePath := filepath.Join("protos", "examples.proto")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("expected --examples to create %s: %v", examplePath, err)
	}
	if !strings.Contains(string(data), "service ExampleService") {
		t.Fatalf("example proto does not contain expected service:\n%s", data)
	}
}

func runInTempDir(t *testing.T) {
	t.Helper()

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("failed to enter temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}
