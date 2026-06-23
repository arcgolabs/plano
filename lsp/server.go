package lsp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	planomodule "github.com/arcgolabs/plano"
	"github.com/arcgolabs/plano/compiler"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type ServerOptions struct {
	Workspace   *Workspace
	Compiler    *compiler.Compiler
	NewCompiler func() *compiler.Compiler
	Client      protocol.Client
}

type Server struct {
	workspace *Workspace

	mu       sync.RWMutex
	client   protocol.Client
	shutdown bool
}

func NewServer(opts ServerOptions) *Server {
	workspace := opts.Workspace
	if workspace == nil {
		workspace = NewWorkspace(Options{
			Compiler:    opts.Compiler,
			NewCompiler: opts.NewCompiler,
		})
	}
	return &Server{
		workspace: workspace,
		client:    opts.Client,
	}
}

func (s *Server) Workspace() *Workspace {
	return s.workspace
}

func (s *Server) SetClient(client protocol.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = client
}

func (s *Server) Initialize(_ context.Context, _ *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	fullSync := protocol.TextDocumentSyncKindFull
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: ptr(true),
				Change:    &fullSync,
				Save:      &protocol.SaveOptions{},
			},
			HoverProvider:          protocol.Boolean(true),
			DefinitionProvider:     protocol.Boolean(true),
			ReferencesProvider:     protocol.Boolean(true),
			DocumentSymbolProvider: protocol.Boolean(true),
			CodeActionProvider: &protocol.CodeActionOptions{
				CodeActionKinds: []protocol.CodeActionKind{protocol.CodeActionKindQuickFix},
			},
			CompletionProvider:   &protocol.CompletionOptions{},
			FoldingRangeProvider: &protocol.FoldingRangeOptions{},
			RenameProvider:       protocol.Boolean(true),
		},
		ServerInfo: protocol.ServerInfo{
			Name:    "plano",
			Version: protocol.NewOptional(planomodule.Version),
		},
	}, nil
}

func (s *Server) Initialized(context.Context, *protocol.InitializedParams) error {
	return nil
}

func (s *Server) Shutdown(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdown = true
	return nil
}

func (s *Server) Exit(context.Context) error {
	return nil
}

func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	if params == nil {
		return errors.New("missing didOpen params")
	}
	if err := s.workspace.Open(string(params.TextDocument.URI), params.TextDocument.Version, []byte(params.TextDocument.Text)); err != nil {
		return err
	}
	return s.publishSnapshotDiagnostics(ctx, params.TextDocument.URI)
}

func (s *Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	if params == nil {
		return errors.New("missing didChange params")
	}
	text, ok, err := fullTextFromChanges(params.ContentChanges)
	if err != nil || !ok {
		return err
	}
	if err := s.workspace.Update(string(params.TextDocument.URI), params.TextDocument.Version, []byte(text)); err != nil {
		return err
	}
	return s.publishSnapshotDiagnostics(ctx, params.TextDocument.URI)
}

func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	if params == nil {
		return errors.New("missing didClose params")
	}
	if err := s.workspace.Close(string(params.TextDocument.URI)); err != nil {
		return err
	}
	return s.publishDiagnostics(ctx, params.TextDocument.URI, 0, nil)
}

func (s *Server) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	if params == nil {
		return errors.New("missing didSave params")
	}
	return s.publishSnapshotDiagnostics(ctx, params.TextDocument.URI)
}

func (s *Server) Handler() jsonrpc2.Handler {
	return protocol.Handlers(s.handleRPC)
}

func (s *Server) handleRPC(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	if result, handled, err := s.handleLifecycle(ctx, req); handled {
		return result, err
	}
	if result, handled, err := s.handleTextDocument(ctx, req); handled {
		return result, err
	}
	result, err := jsonrpc2.MethodNotFoundHandler(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("handle %q: %w", req.Method(), err)
	}
	return result, nil
}

func (s *Server) handleLifecycle(ctx context.Context, req *jsonrpc2.Request) (any, bool, error) {
	switch req.Method() {
	case protocol.MethodInitialize:
		var params protocol.InitializeParams
		result, err := replyCall(ctx, req, &params, s.Initialize)
		return result, true, err
	case protocol.MethodInitialized:
		var params protocol.InitializedParams
		result, err := replyNotify(ctx, req, &params, s.Initialized)
		return result, true, err
	case protocol.MethodShutdown:
		return nil, true, s.Shutdown(ctx)
	case protocol.MethodExit:
		return nil, true, s.Exit(ctx)
	default:
		return nil, false, nil
	}
}

func (s *Server) handleTextDocument(ctx context.Context, req *jsonrpc2.Request) (any, bool, error) {
	if result, handled, err := s.handleTextDocumentSync(ctx, req); handled {
		return result, true, err
	}
	return s.handleTextDocumentQuery(ctx, req)
}

func (s *Server) handleTextDocumentSync(ctx context.Context, req *jsonrpc2.Request) (any, bool, error) {
	switch req.Method() {
	case protocol.MethodTextDocumentDidOpen:
		var params protocol.DidOpenTextDocumentParams
		result, err := replyNotify(ctx, req, &params, s.DidOpen)
		return result, true, err
	case protocol.MethodTextDocumentDidChange:
		var params protocol.DidChangeTextDocumentParams
		result, err := replyNotify(ctx, req, &params, s.DidChange)
		return result, true, err
	case protocol.MethodTextDocumentDidClose:
		var params protocol.DidCloseTextDocumentParams
		result, err := replyNotify(ctx, req, &params, s.DidClose)
		return result, true, err
	case protocol.MethodTextDocumentDidSave:
		var params protocol.DidSaveTextDocumentParams
		result, err := replyNotify(ctx, req, &params, s.DidSave)
		return result, true, err
	default:
		return nil, false, nil
	}
}

func (s *Server) ServeStream(ctx context.Context, conn jsonrpc2.Conn) error {
	if conn == nil {
		return errors.New("nil jsonrpc2 connection")
	}
	s.mu.Lock()
	if s.client == nil {
		s.client = protocol.ClientDispatcher(conn)
	}
	s.mu.Unlock()
	conn.Go(ctx, s.Handler())
	<-conn.Done()
	if err := conn.Err(); err != nil {
		return fmt.Errorf("serve jsonrpc2 connection: %w", err)
	}
	return nil
}
