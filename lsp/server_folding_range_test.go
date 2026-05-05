package lsp_test

import (
	"context"
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"

	"github.com/arcgolabs/plano/lsp"
)

func TestServerInitializeAdvertisesFoldingRanges(t *testing.T) {
	server := lsp.NewServer(lsp.ServerOptions{Workspace: testWorkspace(t)})
	result, err := server.Initialize(context.Background(), &protocol.InitializeParams{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Capabilities.FoldingRangeProvider.(*protocol.FoldingRangeOptions); !ok {
		t.Fatalf("foldingRangeProvider = %#v", result.Capabilities.FoldingRangeProvider)
	}
}

func TestServerFoldingRangesUseWorkspaceState(t *testing.T) {
	ws := testWorkspace(t)
	server := lsp.NewServer(lsp.ServerOptions{Workspace: ws})

	path := filepath.Join(t.TempDir(), "build.plano")
	uri := protocol.DocumentURI(lsp.FileURI(path))
	src := `
workspace {
  name = "demo"
  default = build
}

task build {
  outputs = ["dist/demo"]
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

	ranges, err := server.FoldingRanges(context.Background(), &protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ranges) < 2 {
		t.Fatalf("folding ranges = %#v", ranges)
	}
}
