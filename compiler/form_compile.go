package compiler

import (
	"fmt"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (s *compileState) compileForm(node *ast.FormDecl, locals *env) (*Form, *HIRForm) {
	spec, ok := s.compiler.forms.Get(node.Head.String())
	if !ok {
		s.diags.AddError(node.Pos(), node.End(), fmt.Sprintf("unknown form %q", node.Head.String()))
		return nil, nil
	}

	scopeID := s.scopeID(ScopeForm, node.Pos(), node.End())
	out := &Form{
		Kind:   spec.Name,
		Fields: mapping.NewOrderedMap[string, any](),
		Pos:    node.Pos(),
		End:    node.End(),
	}
	hir := &HIRForm{
		Kind:    spec.Name,
		ScopeID: scopeID,
		Fields:  mapping.NewOrderedMap[string, HIRField](),
		Pos:     node.Pos(),
		End:     node.End(),
	}
	s.applyFormLabel(node, spec, out, hir)

	fieldSeen := map[string]bool{}
	formEnv := s.newScopeEnv(locals, ScopeForm, node.Pos(), node.End())
	if node.Body != nil {
		signal := s.execFormItems(&formExecState{
			spec:      spec,
			form:      out,
			hir:       hir,
			fieldSeen: fieldSeen,
		}, node.Body.Items, formEnv)
		if err := unexpectedLoopControlError(signal); err != nil {
			s.diags.AddError(signal.pos, signal.end, err.Error())
		}
	}
	s.applyFormDefaults(node, spec, out, hir, fieldSeen, scopeID)
	return out, hir
}

func (s *compileState) applyFormLabel(node *ast.FormDecl, spec schema.FormSpec, out *Form, hir *HIRForm) {
	if node.Label != nil {
		out.Label = &FormLabel{
			Kind:  spec.LabelKind,
			Value: node.Label.Value,
		}
		hir.Label = out.Label
	}
	switch spec.LabelKind {
	case schema.LabelNone:
		if node.Label != nil {
			s.diags.AddError(node.Label.Pos(), node.Label.End(), spec.Name+" does not accept label")
		}
	case schema.LabelSymbol:
		s.bindSymbolLabel(node, spec, out, hir)
	case schema.LabelString:
		if node.Label == nil {
			s.diags.AddError(node.Pos(), node.End(), spec.Name+" requires string label")
		} else if !node.Label.Quoted {
			s.diags.AddError(node.Label.Pos(), node.Label.End(), spec.Name+" requires string label")
		}
	}
}

func (s *compileState) bindSymbolLabel(node *ast.FormDecl, spec schema.FormSpec, out *Form, hir *HIRForm) {
	if node.Label == nil {
		s.diags.AddError(node.Pos(), node.End(), spec.Name+" requires symbol label")
		return
	}
	if node.Label.Quoted {
		s.diags.AddError(node.Label.Pos(), node.Label.End(), spec.Name+" requires identifier label")
		return
	}
	if symbol, ok := s.symbols.Get(node.Label.Value); ok {
		sym := symbol
		out.Symbol = &sym
		hir.Symbol = &sym
	}
}

func (s *compileState) applyFormDefaults(
	node *ast.FormDecl,
	spec schema.FormSpec,
	out *Form,
	hir *HIRForm,
	fieldSeen map[string]bool,
	scopeID string,
) {
	for name, field := range spec.Fields {
		if fieldSeen[name] {
			continue
		}
		if field.HasDefault {
			out.Fields.Set(name, field.Default)
			hir.Fields.Set(name, HIRField{
				Name:     name,
				ScopeID:  scopeID,
				Expected: field.Type,
				Actual:   staticTypeOfValue(field.Default),
				Value:    field.Default,
				Pos:      node.Pos(),
				End:      node.End(),
			})
			continue
		}
		if field.Required {
			s.diags.AddError(node.Pos(), node.End(), fmt.Sprintf("%s requires field %q", spec.Name, name))
		}
	}
}

func allowsField(mode schema.BodyMode) bool {
	return mode == schema.BodyFieldOnly || mode == schema.BodyMixed || mode == schema.BodyScript
}

func allowsForm(mode schema.BodyMode) bool {
	return mode == schema.BodyFormOnly || mode == schema.BodyMixed || mode == schema.BodyScript
}

func allowsCall(mode schema.BodyMode) bool {
	return mode == schema.BodyCallOnly || mode == schema.BodyScript
}
