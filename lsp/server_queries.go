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

func (s *Server) handleTextDocumentQuery(ctx context.Context, req *jsonrpc2.Request) (any, bool, error) {
	if result, handled, err := s.handleTextDocumentNavigationQuery(ctx, req); handled {
		return result, true, err
	}
	return s.handleTextDocumentUtilityQuery(ctx, req)
}

func (s *Server) handleTextDocumentNavigationQuery(
	ctx context.Context,
	req *jsonrpc2.Request,
) (any, bool, error) {
	switch req.Method() {
	case protocol.MethodTextDocumentHover:
		var params protocol.HoverParams
		result, err := replyCall(ctx, req, &params, s.Hover)
		return result, true, err
	case protocol.MethodTextDocumentDefinition:
		var params protocol.DefinitionParams
		result, err := replyCall(ctx, req, &params, s.Definition)
		return result, true, err
	case protocol.MethodTextDocumentCompletion:
		var params protocol.CompletionParams
		result, err := replyCall(ctx, req, &params, s.Completion)
		return result, true, err
	case protocol.MethodTextDocumentReferences:
		var params protocol.ReferenceParams
		result, err := replyCall(ctx, req, &params, s.References)
		return result, true, err
	case protocol.MethodTextDocumentPrepareRename:
		var params protocol.PrepareRenameParams
		result, err := replyCall(ctx, req, &params, s.PrepareRename)
		return result, true, err
	case protocol.MethodTextDocumentRename:
		var params protocol.RenameParams
		result, err := replyCall(ctx, req, &params, s.Rename)
		return result, true, err
	default:
		return nil, false, nil
	}
}

func (s *Server) handleTextDocumentUtilityQuery(
	ctx context.Context,
	req *jsonrpc2.Request,
) (any, bool, error) {
	switch req.Method() {
	case protocol.MethodTextDocumentDocumentSymbol:
		var params protocol.DocumentSymbolParams
		result, err := replyCall(ctx, req, &params, s.DocumentSymbols)
		return result, true, err
	case protocol.MethodTextDocumentCodeAction:
		var params protocol.CodeActionParams
		result, err := replyCall(ctx, req, &params, s.CodeAction)
		return result, true, err
	case protocol.MethodTextDocumentFoldingRange:
		var params protocol.FoldingRangeParams
		result, err := replyCall(ctx, req, &params, s.FoldingRanges)
		return result, true, err
	default:
		return nil, false, nil
	}
}

func allowsQuickFix(only []protocol.CodeActionKind) bool {
	if len(only) == 0 {
		return true
	}
	return slices.Contains(only, protocol.CodeActionKindQuickFix)
}
