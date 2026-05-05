package lsp

import (
	"context"
	"errors"
	"slices"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (s *Server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	if params == nil {
		return nil, errors.New("missing hover params")
	}
	snapshot, err := s.workspace.Analyze(ctx, string(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	hover, ok := snapshot.HoverAt(fromProtocolPosition(params.Position))
	if !ok {
		return &protocol.Hover{}, nil
	}
	return toProtocolHover(hover), nil
}

func (s *Server) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	if params == nil {
		return nil, errors.New("missing definition params")
	}
	snapshot, err := s.workspace.Analyze(ctx, string(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	location, ok := snapshot.DefinitionAt(fromProtocolPosition(params.Position))
	if !ok {
		return []protocol.Location{}, nil
	}
	return []protocol.Location{toProtocolLocation(location)}, nil
}

func (s *Server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	if params == nil {
		return nil, errors.New("missing completion params")
	}
	snapshot, err := s.workspace.Analyze(ctx, string(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	completions, ok := snapshot.CompletionAt(fromProtocolPosition(params.Position))
	if !ok {
		return &protocol.CompletionList{}, nil
	}
	return toProtocolCompletionList(completions), nil
}

func (s *Server) References(ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	if params == nil {
		return nil, errors.New("missing references params")
	}
	snapshot, err := s.workspace.Analyze(ctx, string(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	locations, ok := snapshot.ReferencesAt(fromProtocolPosition(params.Position), params.Context.IncludeDeclaration)
	if !ok {
		return []protocol.Location{}, nil
	}
	return toProtocolLocations(locations), nil
}

func (s *Server) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (*protocol.Range, error) {
	if params == nil {
		return nil, errors.New("missing prepareRename params")
	}
	snapshot, err := s.workspace.Analyze(ctx, string(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	rng, ok := snapshot.PrepareRenameAt(fromProtocolPosition(params.Position))
	if !ok {
		return &protocol.Range{}, nil
	}
	protocolRange := toProtocolRange(rng)
	return &protocolRange, nil
}

func (s *Server) Rename(ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	if params == nil {
		return nil, errors.New("missing rename params")
	}
	snapshot, err := s.workspace.Analyze(ctx, string(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	edit, ok := snapshot.RenameAt(fromProtocolPosition(params.Position), params.NewName)
	if !ok {
		return &protocol.WorkspaceEdit{}, nil
	}
	return toProtocolWorkspaceEdit(edit), nil
}

func (s *Server) DocumentSymbols(ctx context.Context, params *protocol.DocumentSymbolParams) ([]any, error) {
	if params == nil {
		return nil, errors.New("missing document symbol params")
	}
	snapshot, err := s.workspace.Analyze(ctx, string(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	return toProtocolDocumentSymbolInterfaces(snapshot.DocumentSymbols()), nil
}

func (s *Server) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	if params == nil {
		return nil, errors.New("missing codeAction params")
	}
	if !allowsQuickFix(params.Context.Only) {
		return []protocol.CodeAction{}, nil
	}
	snapshot, err := s.workspace.Analyze(ctx, string(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	return toProtocolCodeActions(snapshot.CodeActions(fromProtocolRange(params.Range))), nil
}

func (s *Server) FoldingRanges(ctx context.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	if params == nil {
		return nil, errors.New("missing foldingRange params")
	}
	snapshot, err := s.workspace.Analyze(ctx, string(params.TextDocument.URI))
	if err != nil {
		return nil, err
	}
	return toProtocolFoldingRanges(snapshot.FoldingRanges()), nil
}

func (s *Server) handleTextDocumentQuery(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) (bool, error) {
	if handled, err := s.handleTextDocumentNavigationQuery(ctx, reply, req); handled {
		return true, err
	}
	return s.handleTextDocumentUtilityQuery(ctx, reply, req)
}

func (s *Server) handleTextDocumentNavigationQuery(
	ctx context.Context,
	reply jsonrpc2.Replier,
	req jsonrpc2.Request,
) (bool, error) {
	switch req.Method() {
	case protocol.MethodTextDocumentHover:
		var params protocol.HoverParams
		return true, replyCall(ctx, reply, req, &params, s.Hover)
	case protocol.MethodTextDocumentDefinition:
		var params protocol.DefinitionParams
		return true, replyCall(ctx, reply, req, &params, s.Definition)
	case protocol.MethodTextDocumentCompletion:
		var params protocol.CompletionParams
		return true, replyCall(ctx, reply, req, &params, s.Completion)
	case protocol.MethodTextDocumentReferences:
		var params protocol.ReferenceParams
		return true, replyCall(ctx, reply, req, &params, s.References)
	case protocol.MethodTextDocumentPrepareRename:
		var params protocol.PrepareRenameParams
		return true, replyCall(ctx, reply, req, &params, s.PrepareRename)
	case protocol.MethodTextDocumentRename:
		var params protocol.RenameParams
		return true, replyCall(ctx, reply, req, &params, s.Rename)
	default:
		return false, nil
	}
}

func (s *Server) handleTextDocumentUtilityQuery(
	ctx context.Context,
	reply jsonrpc2.Replier,
	req jsonrpc2.Request,
) (bool, error) {
	switch req.Method() {
	case protocol.MethodTextDocumentDocumentSymbol:
		var params protocol.DocumentSymbolParams
		return true, replyCall(ctx, reply, req, &params, s.DocumentSymbols)
	case protocol.MethodTextDocumentCodeAction:
		var params protocol.CodeActionParams
		return true, replyCall(ctx, reply, req, &params, s.CodeAction)
	case protocol.MethodTextDocumentFoldingRange:
		var params protocol.FoldingRangeParams
		return true, replyCall(ctx, reply, req, &params, s.FoldingRanges)
	default:
		return false, nil
	}
}

func allowsQuickFix(only []protocol.CodeActionKind) bool {
	if len(only) == 0 {
		return true
	}
	return slices.Contains(only, protocol.QuickFix)
}
