package lsp

import (
	"errors"
	"fmt"
	"path/filepath"

	lspuri "go.lsp.dev/uri"
)

func PathFromURI(uri string) (string, error) {
	if uri == "" {
		return "", errors.New("empty document uri")
	}
	parsed, err := lspuri.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse uri %q: %w", uri, err)
	}
	if parsed.Scheme() != "file" {
		return "", fmt.Errorf("unsupported uri %q", uri)
	}
	return filepath.Clean(parsed.FsPath()), nil
}

func FileURI(path string) string {
	return string(lspuri.File(path))
}
