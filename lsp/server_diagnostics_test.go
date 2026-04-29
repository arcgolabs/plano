package lsp_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/plano/lsp"
	"go.lsp.dev/protocol"
)

func TestServerPublishesDiagnosticRelatedInformation(t *testing.T) {
	ws := testWorkspace(t)
	client := &recordingClient{}
	server := lsp.NewServer(lsp.ServerOptions{
		Workspace: ws,
		Client:    client,
	})

	uri := protocol.DocumentURI(lsp.FileURI(filepath.Join(t.TempDir(), "dup.plano")))
	src := `
const target: string = "dist"
const target: string = "release"
`
	err := server.DidOpen(context.Background(), &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:     uri,
			Version: 1,
			Text:    src,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(client.diagnostics) != 1 || len(client.diagnostics[0].Diagnostics) == 0 {
		t.Fatalf("published diagnostics = %#v", client.diagnostics)
	}
	item := client.diagnostics[0].Diagnostics[0]
	if item.Code != "duplicate-definition" {
		t.Fatalf("diagnostic code = %#v", item.Code)
	}
	if len(item.RelatedInformation) != 1 {
		t.Fatalf("diagnostic related info = %#v", item.RelatedInformation)
	}
}
