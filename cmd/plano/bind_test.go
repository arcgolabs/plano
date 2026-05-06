package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestBindCommand(t *testing.T) {
	file := writeTempPlano(t, `
const target: string = "dist/demo"

fn output(name: string): path {
  return name
}
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"bind", file})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"Functions"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"output"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %s", stderr.String())
	}
}
