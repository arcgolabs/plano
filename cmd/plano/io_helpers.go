package main

import (
	"fmt"
	"io"
	"os"
)

func readFile(path string) ([]byte, error) {
	//nolint:gosec // The CLI intentionally reads user-provided plano source paths.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	return data, nil
}

func createFile(path string) (*os.File, error) {
	//nolint:gosec // The CLI intentionally writes to user-provided output paths.
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create %q: %w", path, err)
	}
	return file, nil
}

func readOutputFile(path string) ([]byte, error) {
	//nolint:gosec // Tests intentionally read back files they wrote in temporary directories.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read output %q: %w", path, err)
	}
	return data, nil
}

func closeFile(file *os.File) error {
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %q: %w", file.Name(), err)
	}
	return nil
}

func writeString(w io.Writer, text string) error {
	if _, err := io.WriteString(w, text); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func writeBytes(w io.Writer, data []byte) error {
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
