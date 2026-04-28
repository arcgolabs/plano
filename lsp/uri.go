package lsp

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	lspuri "go.lsp.dev/uri"
)

func PathFromURI(uri string) (string, error) {
	if uri == "" {
		return "", errors.New("empty document uri")
	}
	parsed := lspuri.New(uri)
	if !strings.HasPrefix(string(parsed), lspuri.FileScheme+"://") {
		return "", fmt.Errorf("unsupported uri %q", uri)
	}
	return filepath.Clean(parsed.Filename()), nil
}

func FileURI(path string) string {
	return string(lspuri.File(path))
}
