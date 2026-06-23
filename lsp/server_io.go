package lsp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func ServeStdio(ctx context.Context, opts ServerOptions) error {
	conn := jsonrpc2.NewConn(jsonrpc2.NewStream(stdioReadWriteCloser{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}))
	return NewServer(opts).ServeStream(ctx, conn)
}

func (s *Server) publishSnapshotDiagnostics(ctx context.Context, documentURI uri.URI) error {
	snapshot, err := s.workspace.Analyze(ctx, string(documentURI))
	if err != nil {
		return err
	}
	return s.publishDiagnostics(ctx, documentURI, uint32(max(snapshot.Version, 0)), toProtocolDiagnostics(snapshot.Diagnostics))
}

func (s *Server) publishDiagnostics(ctx context.Context, documentURI uri.URI, version uint32, diagnostics []protocol.Diagnostic) error {
	client := s.currentClient()
	if client == nil {
		return nil
	}
	const maxProtocolVersion = uint32(1<<31 - 1)
	if version > maxProtocolVersion {
		version = maxProtocolVersion
	}
	if err := client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         documentURI,
		Version:     protocol.NewOptional(int32(version)),
		Diagnostics: diagnostics,
	}); err != nil {
		return fmt.Errorf("publish diagnostics for %q: %w", documentURI, err)
	}
	return nil
}

func (s *Server) currentClient() protocol.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

func fullTextFromChanges(changes []protocol.TextDocumentContentChangeEvent) (string, bool, error) {
	if len(changes) == 0 {
		return "", false, nil
	}
	switch change := changes[len(changes)-1].(type) {
	case *protocol.TextDocumentContentChangeWholeDocument:
		return change.Text, true, nil
	case *protocol.TextDocumentContentChangePartial:
		return "", false, errors.New("incremental text document changes are not supported")
	default:
		return "", false, errors.New("unsupported text document change event")
	}
}

type stdioReadWriteCloser struct {
	io.Reader
	io.Writer
}

func (stdioReadWriteCloser) Close() error {
	return nil
}
