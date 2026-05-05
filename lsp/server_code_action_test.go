package lsp_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"

	"github.com/arcgolabs/plano/lsp"
)

func TestServerInitializeAdvertisesCodeActions(t *testing.T) {
	server := lsp.NewServer(lsp.ServerOptions{Workspace: testWorkspace(t)})
	result, err := server.Initialize(context.Background(), &protocol.InitializeParams{})
	if err != nil {
		t.Fatal(err)
	}
	options, ok := result.Capabilities.CodeActionProvider.(*protocol.CodeActionOptions)
	if !ok {
		t.Fatalf("codeActionProvider = %#v", result.Capabilities.CodeActionProvider)
	}
	if len(options.CodeActionKinds) != 1 || options.CodeActionKinds[0] != protocol.QuickFix {
		t.Fatalf("code action kinds = %#v", options.CodeActionKinds)
	}
}

func TestServerCodeActionUsesWorkspaceState(t *testing.T) {
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
  outputs = [join_pat("dist", "demo")]
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

	actions, err := server.CodeAction(context.Background(), &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: toProtocolPosition(positionOf(src, "join_pat")),
			End: toProtocolPosition(positionForOffset(
				[]byte(src),
				strings.Index(src, "join_pat")+len("join_pat"),
			)),
		},
		Context: protocol.CodeActionContext{
			Only: []protocol.CodeActionKind{protocol.QuickFix},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) == 0 {
		t.Fatal("expected code actions")
	}
	if actions[0].Title != `Replace with "join_path"` {
		t.Fatalf("actions = %#v", actions)
	}
	if actions[0].Edit == nil || len(actions[0].Edit.Changes) == 0 {
		t.Fatalf("action edit = %#v", actions[0].Edit)
	}
}
