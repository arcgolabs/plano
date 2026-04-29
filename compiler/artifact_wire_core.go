package compiler

import "github.com/arcgolabs/collectionx/list"

func (a Artifact) wire() (artifactWire, error) {
	document, err := artifactDocumentToWire(a.Document)
	if err != nil {
		return artifactWire{}, err
	}
	binding, err := artifactBindingToWire(a.Binding)
	if err != nil {
		return artifactWire{}, err
	}
	checks, err := artifactChecksToWire(a.Checks)
	if err != nil {
		return artifactWire{}, err
	}
	hir, err := artifactHIRToWire(a.HIR)
	if err != nil {
		return artifactWire{}, err
	}
	return artifactWire{
		Document:    document,
		Binding:     binding,
		Checks:      checks,
		HIR:         hir,
		Diagnostics: a.Diagnostics.Values(),
	}, nil
}

func (w artifactWire) artifact() (Artifact, error) {
	document, err := w.Document.artifact()
	if err != nil {
		return Artifact{}, err
	}
	binding, err := w.Binding.artifact()
	if err != nil {
		return Artifact{}, err
	}
	checks, err := w.Checks.artifact()
	if err != nil {
		return Artifact{}, err
	}
	hir, err := w.HIR.artifact()
	if err != nil {
		return Artifact{}, err
	}
	diagnostics := list.NewListWithCapacity[ArtifactDiagnostic](len(w.Diagnostics), w.Diagnostics...)
	return Artifact{
		Document:    document,
		Binding:     binding,
		Checks:      checks,
		HIR:         hir,
		Diagnostics: *diagnostics,
	}, nil
}

func artifactDocumentToWire(value *ArtifactDocument) (*artifactDocumentWire, error) {
	if value == nil {
		return &artifactDocumentWire{}, nil
	}
	forms, err := encodeArtifactWireList(value.Forms, artifactFormToWire)
	if err != nil {
		return nil, err
	}
	symbols, err := encodeArtifactWireMap(value.Symbols, identityArtifactSymbol)
	if err != nil {
		return nil, err
	}
	consts, err := encodeArtifactWireMap(value.Consts, artifactValueToWire)
	if err != nil {
		return nil, err
	}
	return &artifactDocumentWire{
		Forms:   forms,
		Symbols: symbols,
		Consts:  consts,
	}, nil
}

func (w *artifactDocumentWire) artifact() (*ArtifactDocument, error) {
	forms, err := decodeArtifactWireList(wireSlice(w).Forms, artifactFormFromWire)
	if err != nil {
		return nil, err
	}
	symbols, err := decodeArtifactWireMap(wireSlice(w).Symbols, identityArtifactSymbol)
	if err != nil {
		return nil, err
	}
	consts, err := decodeArtifactWireMap(wireSlice(w).Consts, artifactValueFromWire)
	if err != nil {
		return nil, err
	}
	return &ArtifactDocument{
		Forms:   forms,
		Symbols: symbols,
		Consts:  consts,
	}, nil
}

func artifactFormToWire(value ArtifactForm) (artifactFormWire, error) {
	fields, err := encodeArtifactWireMap(value.Fields, artifactValueToWire)
	if err != nil {
		return artifactFormWire{}, err
	}
	forms, err := encodeArtifactWireList(value.Forms, artifactFormToWire)
	if err != nil {
		return artifactFormWire{}, err
	}
	calls, err := encodeArtifactWireList(value.Calls, artifactCallToWire)
	if err != nil {
		return artifactFormWire{}, err
	}
	return artifactFormWire{
		Kind:   value.Kind,
		Label:  value.Label,
		Symbol: value.Symbol,
		Fields: fields,
		Forms:  forms,
		Calls:  calls,
		Span:   value.Span,
	}, nil
}

func artifactFormFromWire(w artifactFormWire) (ArtifactForm, error) {
	fields, err := decodeArtifactWireMap(w.Fields, artifactValueFromWire)
	if err != nil {
		return ArtifactForm{}, err
	}
	forms, err := decodeArtifactWireList(w.Forms, artifactFormFromWire)
	if err != nil {
		return ArtifactForm{}, err
	}
	calls, err := decodeArtifactWireList(w.Calls, artifactCallFromWire)
	if err != nil {
		return ArtifactForm{}, err
	}
	return ArtifactForm{
		Kind:   w.Kind,
		Label:  w.Label,
		Symbol: w.Symbol,
		Fields: fields,
		Forms:  forms,
		Calls:  calls,
		Span:   w.Span,
	}, nil
}

func artifactCallToWire(value ArtifactCall) (artifactCallWire, error) {
	args, err := encodeArtifactWireList(value.Args, artifactValueToWire)
	if err != nil {
		return artifactCallWire{}, err
	}
	return artifactCallWire{Name: value.Name, Args: args, Span: value.Span}, nil
}

func artifactCallFromWire(w artifactCallWire) (ArtifactCall, error) {
	args, err := decodeArtifactWireList(w.Args, artifactValueFromWire)
	if err != nil {
		return ArtifactCall{}, err
	}
	return ArtifactCall{Name: w.Name, Args: args, Span: w.Span}, nil
}

func wireSlice[T any](value *T) T {
	var zero T
	if value == nil {
		return zero
	}
	return *value
}
