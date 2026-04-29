package compiler

func artifactHIRToWire(value *ArtifactHIR) (*artifactHIRWire, error) {
	if value == nil {
		return &artifactHIRWire{}, nil
	}
	forms, err := encodeArtifactWireList(value.Forms, artifactHIRFormToWire)
	if err != nil {
		return nil, err
	}
	symbols, err := encodeArtifactWireMap(value.Symbols, identityArtifactSymbol)
	if err != nil {
		return nil, err
	}
	consts, err := encodeArtifactWireMap(value.Consts, artifactHIRConstToWire)
	if err != nil {
		return nil, err
	}
	return &artifactHIRWire{Forms: forms, Symbols: symbols, Consts: consts}, nil
}

func (w *artifactHIRWire) artifact() (*ArtifactHIR, error) {
	forms, err := decodeArtifactWireList(wireSlice(w).Forms, artifactHIRFormFromWire)
	if err != nil {
		return nil, err
	}
	symbols, err := decodeArtifactWireMap(wireSlice(w).Symbols, identityArtifactSymbol)
	if err != nil {
		return nil, err
	}
	consts, err := decodeArtifactWireMap(wireSlice(w).Consts, artifactHIRConstFromWire)
	if err != nil {
		return nil, err
	}
	return &ArtifactHIR{Forms: forms, Symbols: symbols, Consts: consts}, nil
}

func artifactHIRConstToWire(value ArtifactHIRConst) (artifactHIRConstWire, error) {
	item, err := artifactValueToWire(value.Value)
	if err != nil {
		return artifactHIRConstWire{}, err
	}
	return artifactHIRConstWire{Name: value.Name, Type: value.Type, Value: item, Span: value.Span}, nil
}

func artifactHIRConstFromWire(w artifactHIRConstWire) (ArtifactHIRConst, error) {
	value, err := artifactValueFromWire(w.Value)
	if err != nil {
		return ArtifactHIRConst{}, err
	}
	return ArtifactHIRConst{Name: w.Name, Type: w.Type, Value: value, Span: w.Span}, nil
}

func artifactHIRFormToWire(value ArtifactHIRForm) (artifactHIRFormWire, error) {
	fields, err := encodeArtifactWireMap(value.Fields, artifactHIRFieldToWire)
	if err != nil {
		return artifactHIRFormWire{}, err
	}
	forms, err := encodeArtifactWireList(value.Forms, artifactHIRFormToWire)
	if err != nil {
		return artifactHIRFormWire{}, err
	}
	calls, err := encodeArtifactWireList(value.Calls, artifactHIRCallToWire)
	if err != nil {
		return artifactHIRFormWire{}, err
	}
	return artifactHIRFormWire{
		Kind:    value.Kind,
		ScopeID: value.ScopeID,
		Label:   value.Label,
		Symbol:  value.Symbol,
		Fields:  fields,
		Forms:   forms,
		Calls:   calls,
		Span:    value.Span,
	}, nil
}

func artifactHIRFormFromWire(w artifactHIRFormWire) (ArtifactHIRForm, error) {
	fields, err := decodeArtifactWireMap(w.Fields, artifactHIRFieldFromWire)
	if err != nil {
		return ArtifactHIRForm{}, err
	}
	forms, err := decodeArtifactWireList(w.Forms, artifactHIRFormFromWire)
	if err != nil {
		return ArtifactHIRForm{}, err
	}
	calls, err := decodeArtifactWireList(w.Calls, artifactHIRCallFromWire)
	if err != nil {
		return ArtifactHIRForm{}, err
	}
	return ArtifactHIRForm{
		Kind:    w.Kind,
		ScopeID: w.ScopeID,
		Label:   w.Label,
		Symbol:  w.Symbol,
		Fields:  fields,
		Forms:   forms,
		Calls:   calls,
		Span:    w.Span,
	}, nil
}

func artifactHIRFieldToWire(value ArtifactHIRField) (artifactHIRFieldWire, error) {
	item, err := artifactValueToWire(value.Value)
	if err != nil {
		return artifactHIRFieldWire{}, err
	}
	return artifactHIRFieldWire{
		Name:     value.Name,
		ScopeID:  value.ScopeID,
		Expected: value.Expected,
		Actual:   value.Actual,
		Value:    item,
		Span:     value.Span,
	}, nil
}

func artifactHIRFieldFromWire(w artifactHIRFieldWire) (ArtifactHIRField, error) {
	value, err := artifactValueFromWire(w.Value)
	if err != nil {
		return ArtifactHIRField{}, err
	}
	return ArtifactHIRField{
		Name:     w.Name,
		ScopeID:  w.ScopeID,
		Expected: w.Expected,
		Actual:   w.Actual,
		Value:    value,
		Span:     w.Span,
	}, nil
}

func artifactHIRCallToWire(value ArtifactHIRCall) (artifactHIRCallWire, error) {
	args, err := encodeArtifactWireList(value.Args, artifactHIRArgToWire)
	if err != nil {
		return artifactHIRCallWire{}, err
	}
	return artifactHIRCallWire{
		Name:    value.Name,
		ScopeID: value.ScopeID,
		Args:    args,
		Result:  value.Result,
		Span:    value.Span,
	}, nil
}

func artifactHIRCallFromWire(w artifactHIRCallWire) (ArtifactHIRCall, error) {
	args, err := decodeArtifactWireList(w.Args, artifactHIRArgFromWire)
	if err != nil {
		return ArtifactHIRCall{}, err
	}
	return ArtifactHIRCall{
		Name:    w.Name,
		ScopeID: w.ScopeID,
		Args:    args,
		Result:  w.Result,
		Span:    w.Span,
	}, nil
}

func artifactHIRArgToWire(value ArtifactHIRArg) (artifactHIRArgWire, error) {
	item, err := artifactValueToWire(value.Value)
	if err != nil {
		return artifactHIRArgWire{}, err
	}
	return artifactHIRArgWire{Type: value.Type, Value: item}, nil
}

func artifactHIRArgFromWire(w artifactHIRArgWire) (ArtifactHIRArg, error) {
	value, err := artifactValueFromWire(w.Value)
	if err != nil {
		return ArtifactHIRArg{}, err
	}
	return ArtifactHIRArg{Type: w.Type, Value: value}, nil
}
