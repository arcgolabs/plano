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
	"go.uber.org/zap"
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
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
				Save:      &protocol.SaveOptions{},
			},
			HoverProvider:          true,
			DefinitionProvider:     true,
			ReferencesProvider:     true,
			DocumentSymbolProvider: true,
			CodeActionProvider: &protocol.CodeActionOptions{
				CodeActionKinds: []protocol.CodeActionKind{protocol.QuickFix},
			},
			CompletionProvider: &protocol.CompletionOptions{},
			RenameProvider:     true,
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    "plano",
			Version: planomodule.Version,
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

func (s *Server) handleRPC(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	if handled, err := s.handleLifecycle(ctx, reply, req); handled {
		return err
	}
	if handled, err := s.handleTextDocument(ctx, reply, req); handled {
		return err
	}
	if err := jsonrpc2.MethodNotFoundHandler(ctx, reply, req); err != nil {
		return fmt.Errorf("handle %q: %w", req.Method(), err)
	}
	return nil
}

func (s *Server) handleLifecycle(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) (bool, error) {
	switch req.Method() {
	case protocol.MethodInitialize:
		var params protocol.InitializeParams
		return true, replyCall(ctx, reply, req, &params, s.Initialize)
	case protocol.MethodInitialized:
		var params protocol.InitializedParams
		return true, replyNotify(ctx, reply, req, &params, s.Initialized)
	case protocol.MethodShutdown:
		return true, reply(ctx, nil, s.Shutdown(ctx))
	case protocol.MethodExit:
		return true, reply(ctx, nil, s.Exit(ctx))
	default:
		return false, nil
	}
}

func (s *Server) handleTextDocument(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) (bool, error) {
	if handled, err := s.handleTextDocumentSync(ctx, reply, req); handled {
		return true, err
	}
	return s.handleTextDocumentQuery(ctx, reply, req)
}

func (s *Server) handleTextDocumentSync(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) (bool, error) {
	switch req.Method() {
	case protocol.MethodTextDocumentDidOpen:
		var params protocol.DidOpenTextDocumentParams
		return true, replyNotify(ctx, reply, req, &params, s.DidOpen)
	case protocol.MethodTextDocumentDidChange:
		var params protocol.DidChangeTextDocumentParams
		return true, replyNotify(ctx, reply, req, &params, s.DidChange)
	case protocol.MethodTextDocumentDidClose:
		var params protocol.DidCloseTextDocumentParams
		return true, replyNotify(ctx, reply, req, &params, s.DidClose)
	case protocol.MethodTextDocumentDidSave:
		var params protocol.DidSaveTextDocumentParams
		return true, replyNotify(ctx, reply, req, &params, s.DidSave)
	default:
		return false, nil
	}
}

func (s *Server) ServeStream(ctx context.Context, conn jsonrpc2.Conn) error {
	if conn == nil {
		return errors.New("nil jsonrpc2 connection")
	}
	s.mu.Lock()
	if s.client == nil {
		s.client = protocol.ClientDispatcher(conn, zap.NewNop())
	}
	s.mu.Unlock()
	conn.Go(ctx, s.Handler())
	<-conn.Done()
	if err := conn.Err(); err != nil {
		return fmt.Errorf("serve jsonrpc2 connection: %w", err)
	}
	return nil
}
