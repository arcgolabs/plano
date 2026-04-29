package lsp_test

import (
	"context"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"

	"github.com/arcgolabs/plano/lsp"
)

func TestServerDidOpenPublishesDiagnostics(t *testing.T) {
	ws := testWorkspace(t)
	client := &recordingClient{}
	server := lsp.NewServer(lsp.ServerOptions{
		Workspace: ws,
		Client:    client,
	})

	uri := protocol.DocumentURI(lsp.FileURI(filepath.Join(t.TempDir(), "build.plano")))
	src := `workspace { name = 1 }`
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
	if len(client.diagnostics) != 1 {
		t.Fatalf("published = %d, want 1", len(client.diagnostics))
	}
	if got := client.diagnostics[0]; len(got.Diagnostics) == 0 {
		t.Fatalf("diagnostics = %#v", got.Diagnostics)
	}
}

func TestServerHoverAndDefinitionUseWorkspaceState(t *testing.T) {
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
  let target = join_path("dist", "demo")
  outputs = [target]
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

	hover, err := server.Hover(context.Background(), &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     toProtocolPosition(positionOf(src, "join_path")),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if hover == nil || !strings.Contains(hover.Contents.Value, "fn join_path") {
		t.Fatalf("hover = %#v", hover)
	}

	definition, err := server.Definition(context.Background(), &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     toProtocolPosition(positionOfLast(src, "target")),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(definition) != 1 {
		t.Fatalf("definition = %#v", definition)
	}
	if definition[0].URI != uri {
		t.Fatalf("definition uri = %q, want %q", definition[0].URI, uri)
	}
}

func TestServerCompletionUsesWorkspaceState(t *testing.T) {
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
  let target = jo
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

	offset := strings.Index(src, "jo") + len("jo")
	items, err := server.Completion(context.Background(), &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     toProtocolPosition(positionForOffset([]byte(src), offset)),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if items == nil || len(items.Items) == 0 {
		t.Fatalf("completion = %#v", items)
	}
	if !containsProtocolCompletionLabel(items.Items, "join_path") {
		t.Fatalf("completion items = %#v", items.Items)
	}
}

type recordingClient struct {
	diagnostics []protocol.PublishDiagnosticsParams
}

func (c *recordingClient) Progress(context.Context, *protocol.ProgressParams) error {
	return nil
}

func (c *recordingClient) WorkDoneProgressCreate(context.Context, *protocol.WorkDoneProgressCreateParams) error {
	return nil
}

func (c *recordingClient) LogMessage(context.Context, *protocol.LogMessageParams) error {
	return nil
}

func (c *recordingClient) PublishDiagnostics(_ context.Context, params *protocol.PublishDiagnosticsParams) error {
	if params != nil {
		c.diagnostics = append(c.diagnostics, *params)
	}
	return nil
}

func (c *recordingClient) ShowMessage(context.Context, *protocol.ShowMessageParams) error {
	return nil
}

func (c *recordingClient) ShowMessageRequest(context.Context, *protocol.ShowMessageRequestParams) (*protocol.MessageActionItem, error) {
	return &protocol.MessageActionItem{}, nil
}

func (c *recordingClient) Telemetry(context.Context, any) error {
	return nil
}

func (c *recordingClient) RegisterCapability(context.Context, *protocol.RegistrationParams) error {
	return nil
}

func (c *recordingClient) UnregisterCapability(context.Context, *protocol.UnregistrationParams) error {
	return nil
}

func (c *recordingClient) ApplyEdit(context.Context, *protocol.ApplyWorkspaceEditParams) (bool, error) {
	return false, nil
}

func (c *recordingClient) Configuration(context.Context, *protocol.ConfigurationParams) ([]any, error) {
	return nil, nil
}

func (c *recordingClient) WorkspaceFolders(context.Context) ([]protocol.WorkspaceFolder, error) {
	return nil, nil
}

func toProtocolPosition(pos lsp.Position) protocol.Position {
	return protocol.Position{
		Line:      clampUint32(pos.Line),
		Character: clampUint32(pos.Character),
	}
}

func clampUint32(value int) uint32 {
	switch {
	case value <= 0:
		return 0
	case value >= math.MaxUint32:
		return math.MaxUint32
	default:
		return uint32(value)
	}
}

func containsProtocolCompletionLabel(items []protocol.CompletionItem, want string) bool {
	for index := range items {
		item := items[index]
		if item.Label == want {
			return true
		}
	}
	return false
}
