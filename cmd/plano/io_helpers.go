package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func readFile(path string) ([]byte, error) {
	root, name, err := openPathRoot(path)
	if err != nil {
		return nil, err
	}
	data, err := root.ReadFile(name)
	closeErr := root.Close()
	if err != nil {
		if closeErr != nil {
			return nil, errors.Join(fmt.Errorf("read %q: %w", path, err), fmt.Errorf("close root for %q: %w", path, closeErr))
		}
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close root for %q: %w", path, closeErr)
	}
	return data, nil
}

func openWriter(path string) (io.WriteCloser, error) {
	root, name, err := openPathRoot(path)
	if err != nil {
		return nil, err
	}
	file, err := root.Create(name)
	if err != nil {
		if closeErr := root.Close(); closeErr != nil {
			return nil, errors.Join(fmt.Errorf("create %q: %w", path, err), fmt.Errorf("close root for %q: %w", path, closeErr))
		}
		return nil, fmt.Errorf("create %q: %w", path, err)
	}
	return &rootedWriteCloser{
		file: file,
		root: root,
		path: path,
	}, nil
}

func readOutputFile(path string) ([]byte, error) {
	return readFile(path)
}

func writeString(w io.Writer, text string) error {
	if _, err := io.WriteString(w, text); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

type rootedWriteCloser struct {
	file *os.File
	root *os.Root
	path string
}

func (w *rootedWriteCloser) Write(data []byte) (int, error) {
	n, err := w.file.Write(data)
	if err != nil {
		return n, fmt.Errorf("write %q: %w", w.path, err)
	}
	return n, nil
}

func (w *rootedWriteCloser) Close() error {
	err := errors.Join(w.file.Close(), w.root.Close())
	if err != nil {
		return fmt.Errorf("close %q: %w", w.path, err)
	}
	return nil
}

func openPathRoot(path string) (*os.Root, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("resolve %q: %w", path, err)
	}
	root, err := os.OpenRoot(filepath.Dir(abs))
	if err != nil {
		return nil, "", fmt.Errorf("open root for %q: %w", path, err)
	}
	return root, filepath.Base(abs), nil
}

func writeBytes(w io.Writer, data []byte) error {
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
