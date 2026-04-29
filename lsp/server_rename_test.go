package lsp_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/plano/lsp"
	"go.lsp.dev/protocol"
)

func TestServerPrepareRenameAndRenameUseWorkspaceState(t *testing.T) {
	ws := testWorkspace(t)
	server := lsp.NewServer(lsp.ServerOptions{Workspace: ws})

	path := filepath.Join(t.TempDir(), "build.plano")
	uri := protocol.DocumentURI(lsp.FileURI(path))
	src := `
const project_name: string = "demo"

workspace {
  name = project_name
  default = build
}

task build {
  outputs = [join_path("dist", project_name)]
}
`
	if err := server.DidOpen(context.Background(), &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:     uri,
			Version: 1,
			Text:    src,
		},
	}); err != nil {
		t.Fatal(err)
	}

	position := toProtocolPosition(positionOf(src, "project_name"))
	rng, err := server.PrepareRename(context.Background(), &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rng == nil || rng.Start == rng.End {
		t.Fatalf("prepare rename range = %#v", rng)
	}

	edit, err := server.Rename(context.Background(), &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
		NewName: "project_id",
	})
	if err != nil {
		t.Fatal(err)
	}
	if edit == nil || len(edit.Changes) != 1 {
		t.Fatalf("rename edit = %#v", edit)
	}
	if got := len(edit.Changes[uri]); got != 3 {
		t.Fatalf("rename edits = %d, want 3", got)
	}
}
