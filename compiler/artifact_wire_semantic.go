package compiler

import "github.com/arcgolabs/collectionx/list"

func artifactBindingToWire(value *ArtifactBinding) (*artifactBindingWire, error) {
	if value == nil {
		return &artifactBindingWire{}, nil
	}
	scopes, err := encodeArtifactWireMap(value.Scopes, identityArtifactScope)
	if err != nil {
		return nil, err
	}
	locals, err := encodeArtifactWireMap(value.Locals, identityArtifactLocal)
	if err != nil {
		return nil, err
	}
	uses, err := encodeArtifactWireMap(value.Uses, identityArtifactUse)
	if err != nil {
		return nil, err
	}
	symbols, err := encodeArtifactWireMap(value.Symbols, identityArtifactSymbol)
	if err != nil {
		return nil, err
	}
	consts, err := encodeArtifactWireMap(value.Consts, identityArtifactConst)
	if err != nil {
		return nil, err
	}
	functions, err := encodeArtifactWireMap(value.Functions, artifactFunctionToWire)
	if err != nil {
		return nil, err
	}
	return &artifactBindingWire{
		Files:     value.Files.Values(),
		Scopes:    scopes,
		Locals:    locals,
		Uses:      uses,
		Symbols:   symbols,
		Consts:    consts,
		Functions: functions,
	}, nil
}

func (w *artifactBindingWire) artifact() (*ArtifactBinding, error) {
	files := list.NewListWithCapacity[string](len(wireSlice(w).Files), wireSlice(w).Files...)
	scopes, err := decodeArtifactWireMap(wireSlice(w).Scopes, identityArtifactScope)
	if err != nil {
		return nil, err
	}
	locals, err := decodeArtifactWireMap(wireSlice(w).Locals, identityArtifactLocal)
	if err != nil {
		return nil, err
	}
	uses, err := decodeArtifactWireMap(wireSlice(w).Uses, identityArtifactUse)
	if err != nil {
		return nil, err
	}
	symbols, err := decodeArtifactWireMap(wireSlice(w).Symbols, identityArtifactSymbol)
	if err != nil {
		return nil, err
	}
	consts, err := decodeArtifactWireMap(wireSlice(w).Consts, identityArtifactConst)
	if err != nil {
		return nil, err
	}
	functions, err := decodeArtifactWireMap(wireSlice(w).Functions, artifactFunctionFromWire)
	if err != nil {
		return nil, err
	}
	return &ArtifactBinding{
		Files:     *files,
		Scopes:    scopes,
		Locals:    locals,
		Uses:      uses,
		Symbols:   symbols,
		Consts:    consts,
		Functions: functions,
	}, nil
}

func artifactChecksToWire(value *ArtifactChecks) (*artifactChecksWire, error) {
	if value == nil {
		return &artifactChecksWire{}, nil
	}
	exprs, err := encodeArtifactWireMap(value.Exprs, identityArtifactExprCheck)
	if err != nil {
		return nil, err
	}
	fields, err := encodeArtifactWireMap(value.Fields, identityArtifactFieldCheck)
	if err != nil {
		return nil, err
	}
	calls, err := encodeArtifactWireMap(value.Calls, artifactCallCheckToWire)
	if err != nil {
		return nil, err
	}
	return &artifactChecksWire{Exprs: exprs, Fields: fields, Calls: calls}, nil
}

func (w *artifactChecksWire) artifact() (*ArtifactChecks, error) {
	exprs, err := decodeArtifactWireMap(wireSlice(w).Exprs, identityArtifactExprCheck)
	if err != nil {
		return nil, err
	}
	fields, err := decodeArtifactWireMap(wireSlice(w).Fields, identityArtifactFieldCheck)
	if err != nil {
		return nil, err
	}
	calls, err := decodeArtifactWireMap(wireSlice(w).Calls, artifactCallCheckFromWire)
	if err != nil {
		return nil, err
	}
	return &ArtifactChecks{Exprs: exprs, Fields: fields, Calls: calls}, nil
}

func artifactFunctionToWire(value ArtifactFunction) (artifactFunctionWire, error) {
	return artifactFunctionWire{
		Name:   value.Name,
		Params: value.Params.Values(),
		Result: value.Result,
		Span:   value.Span,
	}, nil
}

func artifactFunctionFromWire(w artifactFunctionWire) (ArtifactFunction, error) {
	params := list.NewListWithCapacity[ArtifactParam](len(w.Params), w.Params...)
	return ArtifactFunction{Name: w.Name, Params: *params, Result: w.Result, Span: w.Span}, nil
}

func artifactCallCheckToWire(value ArtifactCallCheck) (artifactCallCheckWire, error) {
	return artifactCallCheckWire{
		ID:      value.ID,
		Name:    value.Name,
		ScopeID: value.ScopeID,
		Args:    value.Args.Values(),
		Result:  value.Result,
		Span:    value.Span,
	}, nil
}

func artifactCallCheckFromWire(w artifactCallCheckWire) (ArtifactCallCheck, error) {
	args := list.NewListWithCapacity[ArtifactType](len(w.Args), w.Args...)
	return ArtifactCallCheck{
		ID:      w.ID,
		Name:    w.Name,
		ScopeID: w.ScopeID,
		Args:    *args,
		Result:  w.Result,
		Span:    w.Span,
	}, nil
}
