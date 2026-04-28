package lsp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func ServeStdio(ctx context.Context, opts ServerOptions) error {
	conn := jsonrpc2.NewConn(jsonrpc2.NewStream(stdioReadWriteCloser{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}))
	return NewServer(opts).ServeStream(ctx, conn)
}

func (s *Server) publishSnapshotDiagnostics(ctx context.Context, uri protocol.DocumentURI) error {
	snapshot, err := s.workspace.Analyze(ctx, string(uri))
	if err != nil {
		return err
	}
	return s.publishDiagnostics(ctx, uri, uint32(max(snapshot.Version, 0)), toProtocolDiagnostics(snapshot.Diagnostics))
}

func (s *Server) publishDiagnostics(ctx context.Context, uri protocol.DocumentURI, version uint32, diagnostics []protocol.Diagnostic) error {
	client := s.currentClient()
	if client == nil {
		return nil
	}
	if err := client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Version:     version,
		Diagnostics: diagnostics,
	}); err != nil {
		return fmt.Errorf("publish diagnostics for %q: %w", uri, err)
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
	change := changes[len(changes)-1]
	if change.Range != (protocol.Range{}) || change.RangeLength != 0 {
		return "", false, errors.New("incremental text document changes are not supported")
	}
	return change.Text, true, nil
}

type stdioReadWriteCloser struct {
	io.Reader
	io.Writer
}

func (stdioReadWriteCloser) Close() error {
	return nil
}
