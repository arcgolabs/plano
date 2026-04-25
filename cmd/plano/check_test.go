package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheckCommandWithExample(t *testing.T) {
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
	cmd.SetArgs([]string{"check", "--example", "builddsl", file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"checks"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"binding"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}
