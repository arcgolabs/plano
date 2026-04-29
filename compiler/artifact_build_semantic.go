package compiler

import (
	"github.com/arcgolabs/collectionx/list"
)

func (b artifactBuilder) binding(binding *Binding) (*ArtifactBinding, error) {
	if binding == nil {
		return emptyArtifactBinding(), nil
	}
	out := emptyArtifactBinding()
	out.Files = binding.Files
	b.addBindingScopes(out, binding)
	b.addBindingLocals(out, binding)
	b.addBindingUses(out, binding)
	out.Symbols = b.symbolMap(binding.Symbols)
	b.addBindingConsts(out, binding)
	if err := b.addBindingFunctions(out, binding); err != nil {
		return nil, err
	}
	return out, nil
}

func (b artifactBuilder) addBindingScopes(out *ArtifactBinding, binding *Binding) {
	if out == nil || binding == nil || binding.Scopes == nil {
		return
	}
	binding.Scopes.Range(func(id string, scope ScopeBinding) bool {
		out.Scopes.Set(id, ArtifactScope{
			ID:       scope.ID,
			Kind:     scope.Kind,
			FormKind: scope.FormKind,
			ParentID: scope.ParentID,
			Span:     b.span(scope.Pos, scope.End),
		})
		return true
	})
}

func (b artifactBuilder) addBindingLocals(out *ArtifactBinding, binding *Binding) {
	if out == nil || binding == nil || binding.Locals == nil {
		return
	}
	binding.Locals.Range(func(id string, item LocalBinding) bool {
		out.Locals.Set(id, ArtifactLocal{
			ID:      item.ID,
			Name:    item.Name,
			Kind:    item.Kind,
			ScopeID: item.ScopeID,
			Type:    artifactType(item.Type),
			Span:    b.span(item.Pos, item.End),
		})
		return true
	})
}

func (b artifactBuilder) addBindingUses(out *ArtifactBinding, binding *Binding) {
	if out == nil || binding == nil || binding.Uses == nil {
		return
	}
	binding.Uses.Range(func(id string, item NameUse) bool {
		out.Uses.Set(id, ArtifactUse{
			ID:       item.ID,
			Name:     item.Name,
			Kind:     item.Kind,
			ScopeID:  item.ScopeID,
			TargetID: item.TargetID,
			Span:     b.span(item.Pos, item.End),
		})
		return true
	})
}

func (b artifactBuilder) addBindingConsts(out *ArtifactBinding, binding *Binding) {
	if out == nil || binding == nil || binding.Consts == nil {
		return
	}
	binding.Consts.Range(func(name string, item ConstBinding) bool {
		out.Consts.Set(name, ArtifactConst{
			Name: item.Name,
			Type: artifactType(item.Type),
			Span: b.span(item.Pos, item.End),
		})
		return true
	})
}

func (b artifactBuilder) addBindingFunctions(out *ArtifactBinding, binding *Binding) error {
	if out == nil || binding == nil || binding.Functions == nil {
		return nil
	}
	for _, name := range binding.Functions.Keys() {
		item, _ := binding.Functions.Get(name)
		params, err := encodeArtifactList(item.Params, b.param)
		if err != nil {
			return err
		}
		out.Functions.Set(name, ArtifactFunction{
			Name:   item.Name,
			Params: params,
			Result: artifactType(item.Result),
			Span:   b.span(item.Pos, item.End),
		})
	}
	return nil
}

func (b artifactBuilder) param(item ParamBinding) (ArtifactParam, error) {
	return ArtifactParam{
		Name: item.Name,
		Type: artifactType(item.Type),
		Span: b.span(item.Pos, item.End),
	}, nil
}

func (b artifactBuilder) checks(checks *CheckInfo) *ArtifactChecks {
	out := emptyArtifactChecks()
	if checks == nil {
		return out
	}
	b.addExprChecks(out, checks)
	b.addFieldChecks(out, checks)
	b.addCallChecks(out, checks)
	return out
}

func (b artifactBuilder) addExprChecks(out *ArtifactChecks, checks *CheckInfo) {
	if out == nil || checks == nil || checks.Exprs == nil {
		return
	}
	checks.Exprs.Range(func(id string, item ExprCheck) bool {
		out.Exprs.Set(id, ArtifactExprCheck{
			ID:      item.ID,
			Kind:    item.Kind,
			ScopeID: item.ScopeID,
			Type:    artifactType(item.Type),
			Span:    b.span(item.Pos, item.End),
		})
		return true
	})
}

func (b artifactBuilder) addFieldChecks(out *ArtifactChecks, checks *CheckInfo) {
	if out == nil || checks == nil || checks.Fields == nil {
		return
	}
	checks.Fields.Range(func(id string, item FieldCheck) bool {
		out.Fields.Set(id, ArtifactFieldCheck{
			ID:       item.ID,
			FormKind: item.FormKind,
			Field:    item.Field,
			ScopeID:  item.ScopeID,
			Expected: artifactType(item.Expected),
			Actual:   artifactType(item.Actual),
			Span:     b.span(item.Pos, item.End),
		})
		return true
	})
}

func (b artifactBuilder) addCallChecks(out *ArtifactChecks, checks *CheckInfo) {
	if out == nil || checks == nil || checks.Calls == nil {
		return
	}
	for _, id := range checks.Calls.Keys() {
		item, _ := checks.Calls.Get(id)
		args := list.NewListWithCapacity[ArtifactType](item.Args.Len())
		for index := range item.Args.Len() {
			arg, _ := item.Args.Get(index)
			args.Add(artifactType(arg))
		}
		out.Calls.Set(id, ArtifactCallCheck{
			ID:      item.ID,
			Name:    item.Name,
			ScopeID: item.ScopeID,
			Args:    *args,
			Result:  artifactType(item.Result),
			Span:    b.span(item.Pos, item.End),
		})
	}
}
