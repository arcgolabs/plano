package main

import (
	"errors"
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
			return nil, wrapCLIErrorf(errors.Join(err, closeErr), "read and close %q", path)
		}
		return nil, wrapCLIErrorf(err, "read %q", path)
	}
	if closeErr != nil {
		return nil, wrapCLIErrorf(closeErr, "close root for %q", path)
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
			return nil, wrapCLIErrorf(errors.Join(err, closeErr), "create and close %q", path)
		}
		return nil, wrapCLIErrorf(err, "create %q", path)
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
		return wrapCLIErrorf(err, "write output")
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
		return n, wrapCLIErrorf(err, "write %q", w.path)
	}
	return n, nil
}

func (w *rootedWriteCloser) Close() error {
	err := errors.Join(w.file.Close(), w.root.Close())
	if err != nil {
		return wrapCLIErrorf(err, "close %q", w.path)
	}
	return nil
}

func openPathRoot(path string) (*os.Root, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", wrapCLIErrorf(err, "resolve %q", path)
	}
	root, err := os.OpenRoot(filepath.Dir(abs))
	if err != nil {
		return nil, "", wrapCLIErrorf(err, "open root for %q", path)
	}
	return root, filepath.Base(abs), nil
}

func writeBytes(w io.Writer, data []byte) error {
	if _, err := w.Write(data); err != nil {
		return wrapCLIErrorf(err, "write output")
	}
	return nil
}
