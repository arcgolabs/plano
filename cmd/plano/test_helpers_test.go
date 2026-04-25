package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempPlano(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "build.plano")
	if err := os.WriteFile(file, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}
