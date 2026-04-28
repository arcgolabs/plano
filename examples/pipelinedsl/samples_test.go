package pipelinedsl_test

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

func TestLowerAllBundledSamples(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fsys := os.DirFS(".")

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".plano") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			src, err := fs.ReadFile(fsys, name)
			if err != nil {
				t.Fatal(err)
			}
			_ = compilePipeline(t, src)
		})
	}
}
