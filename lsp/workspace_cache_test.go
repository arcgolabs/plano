package lsp_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/examples/builddsl"
	"github.com/arcgolabs/plano/lsp"
)

func TestWorkspaceAnalyzeCachesOpenDocumentSnapshots(t *testing.T) {
	rootDir := t.TempDir()
	defsPath := filepath.Join(rootDir, "defs.plano")
	rootPath := filepath.Join(rootDir, "build.plano")
	rootURI := fileURI(rootPath)

	if err := os.WriteFile(defsPath, []byte(`const project_name: string = "demo"`), 0o600); err != nil {
		t.Fatal(err)
	}

	source := `
import "./defs.plano"

workspace {
  name = project_name
  default = build
}

task build {}
`

	ws, reads := testWorkspaceWithReadCounter(t)
	if err := ws.Open(rootURI, 1, []byte(source)); err != nil {
		t.Fatal(err)
	}

	if _, err := ws.Analyze(context.Background(), rootURI); err != nil {
		t.Fatal(err)
	}
	firstReads := reads.Load()
	if firstReads == 0 {
		t.Fatal("expected imported file reads")
	}

	if _, err := ws.Analyze(context.Background(), rootURI); err != nil {
		t.Fatal(err)
	}
	if reads.Load() != firstReads {
		t.Fatalf("read count = %d, want cached %d", reads.Load(), firstReads)
	}
}

func TestWorkspaceAnalyzeInvalidatesSnapshotCacheOnUpdate(t *testing.T) {
	rootDir := t.TempDir()
	defsPath := filepath.Join(rootDir, "defs.plano")
	rootPath := filepath.Join(rootDir, "build.plano")
	rootURI := fileURI(rootPath)

	if err := os.WriteFile(defsPath, []byte(`const project_name: string = "demo"`), 0o600); err != nil {
		t.Fatal(err)
	}

	source := `
import "./defs.plano"

workspace {
  name = project_name
  default = build
}

task build {}
`

	ws, reads := testWorkspaceWithReadCounter(t)
	if err := ws.Open(rootURI, 1, []byte(source)); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Analyze(context.Background(), rootURI); err != nil {
		t.Fatal(err)
	}
	firstReads := reads.Load()

	updated := source + "\n"
	if err := ws.Update(rootURI, 2, []byte(updated)); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Analyze(context.Background(), rootURI); err != nil {
		t.Fatal(err)
	}
	if reads.Load() <= firstReads {
		t.Fatalf("read count = %d, want > %d after invalidation", reads.Load(), firstReads)
	}
}

type readCounter struct {
	mu    sync.Mutex
	count int
}

func (c *readCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

func (c *readCounter) Load() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

func testWorkspaceWithReadCounter(t *testing.T) (*lsp.Workspace, *readCounter) {
	t.Helper()

	reads := &readCounter{}
	root, err := os.OpenRoot("/")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := root.Close(); closeErr != nil {
			t.Fatalf("close root: %v", closeErr)
		}
	})
	base := compiler.New(compiler.Options{
		ReadFile: func(path string) ([]byte, error) {
			reads.Inc()
			name := strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator))
			return root.ReadFile(name)
		},
	})
	if err := builddsl.Register(base); err != nil {
		t.Fatal(err)
	}
	return lsp.NewWorkspace(lsp.Options{Compiler: base}), reads
}
