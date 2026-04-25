package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	file := writeTempPlano(t, `
const target: string = "dist/demo"
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"parse", file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"Statements"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestParseCommandYAMLOutputToFile(t *testing.T) {
	file := writeTempPlano(t, `
const target: string = "dist/demo"
`)
	outFile := filepath.Join(t.TempDir(), "ast.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"parse", "--format", "yaml", "--out", outFile, file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %s", stdout.String())
	}
	data, err := readOutputFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Statements:") {
		t.Fatalf("file = %s", string(data))
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestCompileCommandWithExample(t *testing.T) {
	file := writeTempPlano(t, `
workspace {
  name = "demo"
  default = build
}

go.binary build {
  main = "./cmd/demo"
  out = "dist/demo"
}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"compile", "--example", "builddsl", file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"Forms"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestValidateCommand(t *testing.T) {
	file := writeTempPlano(t, `
workspace {
  name = "demo"
  default = build
}

go.binary build {
  main = "./cmd/demo"
  out = "dist/demo"
}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"validate", "--example", "builddsl", file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "valid" {
		t.Fatalf("stdout = %q, want valid", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestValidateStrictFailsOnDiagnostics(t *testing.T) {
	file := writeTempPlano(t, `
workspace {
  name = "demo"
  default = missing
}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"validate", "--example", "builddsl", "--strict", file})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), `undefined symbol "missing"`) {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestLowerCommandWithExample(t *testing.T) {
	file := writeTempPlano(t, `
workspace {
  name = "demo"
  default = build
}

go.binary build {
  main = "./cmd/demo"
  out = "dist/demo"
}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"lower", "--example", "builddsl", file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"DefaultTask": "build"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestDiagCommandJSON(t *testing.T) {
	file := writeTempPlano(t, `
workspace {
  name = "demo"
  default = missing
}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"diag", "--example", "builddsl", "--format", "json", file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"severity": "error"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestLowerCommandYAMLOutputToFile(t *testing.T) {
	file := writeTempPlano(t, `
workspace {
  name = "demo"
  default = build
}

go.binary build {
  main = "./cmd/demo"
  out = "dist/demo"
}
`)
	outFile := filepath.Join(t.TempDir(), "project.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"lower", "--example", "builddsl", "--format", "yaml", "--out", outFile, file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %s", stdout.String())
	}
	data, err := readOutputFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "DefaultTask: build") {
		t.Fatalf("file = %s", string(data))
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestLowerRequiresExample(t *testing.T) {
	file := writeTempPlano(t, `workspace {}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"lower", file})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires --example") {
		t.Fatalf("err = %v", err)
	}
}

func writeTempPlano(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "build.plano")
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}
