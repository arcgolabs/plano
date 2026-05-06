package compiler

import (
	"fmt"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

func (s *compileState) execAssignment(state *formExecState, current *ast.Assignment, locals *env) {
	if state.spec.BodyMode == schema.BodyScript && s.shouldAssignLocal(state.spec, current.Name.Name, locals) {
		s.execLocalAssignment(current, locals)
		return
	}
	s.execFieldAssignment(state, current, locals)
}

func (s *compileState) execFieldAssignment(state *formExecState, current *ast.Assignment, locals *env) {
	if !allowsField(state.spec.BodyMode) {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow fields in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	fieldSpec, ok := formFieldSpec(state.spec, current.Name.Name)
	if !ok {
		s.diags.AddErrorCodeSuggestions(
			diag.CodeUnknownField,
			current.Pos(),
			current.End(),
			fmt.Sprintf("field %q is not allowed in %s", current.Name.Name, state.spec.Name),
			s.fieldSuggestions(state.spec, current.Name.Name, current.Name.Pos(), current.Name.End())...,
		)
		return
	}
	value, err := s.evalExpr(current.Value, locals)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return
	}
	if err := schema.CheckAssignable(fieldSpec.Type, value); err != nil {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("field %q: %v", fieldSpec.Name, err))
		return
	}
	state.form.Fields.Set(current.Name.Name, value)
	state.fieldSeen.Add(current.Name.Name)
	scopeID := locals.scope
	expected := fieldSpec.Type
	actual := staticTypeOfValue(value)
	if check, ok := s.fieldCheck(current.Pos(), current.End()); ok {
		scopeID = check.ScopeID
		expected = check.Expected
		actual = check.Actual
	}
	state.hir.Fields.Set(current.Name.Name, HIRField{
		Name:     fieldSpec.Name,
		ScopeID:  scopeID,
		Expected: expected,
		Actual:   actual,
		Value:    value,
		Pos:      current.Pos(),
		End:      current.End(),
	})
}

func (s *compileState) shouldAssignLocal(spec schema.FormSpec, name string, locals *env) bool {
	if hasFormField(spec, name) {
		return false
	}
	_, ok := locals.Lookup(name)
	return ok
}

func (s *compileState) execLocalAssignment(current *ast.Assignment, locals *env) {
	value, err := s.evalExpr(current.Value, locals)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return
	}
	if err := locals.Assign(current.Name.Name, value); err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
	}
}

func (s *compileState) execLocalBinding(state *formExecState, kind LocalBindingKind, name string, typeExpr ast.TypeExpr, expr ast.Expr, locals *env) {
	if state.spec.BodyMode != schema.BodyScript {
		s.diags.AddError(expr.Pos(), expr.End(), fmt.Sprintf("%s does not allow script statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	if hasFormField(state.spec, name) {
		s.diags.AddError(expr.Pos(), expr.End(), fmt.Sprintf("binding %q conflicts with field %q in %s", name, name, state.spec.Name))
		return
	}
	value, err := s.evalExpr(expr, locals)
	if err != nil {
		s.diags.AddError(expr.Pos(), expr.End(), err.Error())
		return
	}
	if err := s.bindLocalValue(locals, kind, name, typeExpr, value); err != nil {
		s.diags.AddError(expr.Pos(), expr.End(), err.Error())
	}
}
