package compiler

import (
	"fmt"
	"go/token"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

type formExecState struct {
	spec      schema.FormSpec
	form      *Form
	hir       *HIRForm
	fieldSeen map[string]bool
}

func (s *compileState) execFormItems(state *formExecState, items []ast.FormItem, locals *env) execSignal {
	for _, item := range items {
		signal, handled := s.execFormStructuralItem(state, item, locals)
		if handled {
			if !signal.IsNone() {
				return signal
			}
			continue
		}
		s.execUnsupportedFormItem(item)
	}
	return noExecSignal()
}

func (s *compileState) execFormStructuralItem(state *formExecState, item ast.FormItem, locals *env) (execSignal, bool) {
	if signal, ok := s.execFormStatementItem(state, item, locals); ok {
		return signal, true
	}
	return s.execFormScriptItem(state, item, locals)
}

func (s *compileState) execFormStatementItem(state *formExecState, item ast.FormItem, locals *env) (execSignal, bool) {
	switch current := item.(type) {
	case *ast.Assignment:
		s.execAssignment(state, current, locals)
	case *ast.FormDecl:
		s.execNestedForm(state, current, locals)
	case *ast.CallStmt:
		s.execCall(state, current, locals)
	default:
		return noExecSignal(), false
	}
	return noExecSignal(), true
}

func (s *compileState) execAssignment(state *formExecState, current *ast.Assignment, locals *env) {
	if state.spec.BodyMode == schema.BodyScript && s.shouldAssignLocal(state.spec, current.Name.Name, locals) {
		s.execLocalAssignment(current, locals)
		return
	}
	s.execFieldAssignment(state, current, locals)
}

func (s *compileState) execFormScriptItem(state *formExecState, item ast.FormItem, locals *env) (execSignal, bool) {
	switch current := item.(type) {
	case *ast.ConstDecl:
		s.execLocalBinding(state, LocalConst, current.Name.Name, current.Type, current.Value, locals)
		return noExecSignal(), true
	case *ast.LetDecl:
		s.execLocalBinding(state, LocalLet, current.Name.Name, current.Type, current.Value, locals)
		return noExecSignal(), true
	case *ast.IfStmt:
		return s.execIf(state, current, locals), true
	case *ast.ForStmt:
		return s.execFor(state, current, locals), true
	case *ast.BreakStmt:
		return s.execFormLoopControl(state, current.Pos(), current.End(), true), true
	case *ast.ContinueStmt:
		return s.execFormLoopControl(state, current.Pos(), current.End(), false), true
	default:
		return noExecSignal(), false
	}
}

func (s *compileState) execUnsupportedFormItem(item ast.FormItem) {
	switch current := item.(type) {
	case *ast.ReturnStmt:
		s.diags.AddError(current.Pos(), current.End(), "return is not allowed in form bodies")
	case *ast.FnDecl:
		s.diags.AddError(current.Pos(), current.End(), "nested function declarations are not implemented")
	case *ast.ImportDecl:
		s.diags.AddError(current.Pos(), current.End(), "import is not allowed in form bodies")
	default:
		s.diags.AddError(item.Pos(), item.End(), fmt.Sprintf("unsupported form item %T", item))
	}
}

func (s *compileState) execFieldAssignment(state *formExecState, current *ast.Assignment, locals *env) {
	if !allowsField(state.spec.BodyMode) {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow fields in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	fieldSpec, ok := state.spec.Fields[current.Name.Name]
	if !ok {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("field %q is not allowed in %s", current.Name.Name, state.spec.Name))
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
	state.fieldSeen[current.Name.Name] = true
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
	if _, ok := spec.Fields[name]; ok {
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

func (s *compileState) execNestedForm(state *formExecState, current *ast.FormDecl, locals *env) {
	if !allowsForm(state.spec.BodyMode) {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow nested forms in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	if len(state.spec.NestedForms) > 0 {
		if _, ok := state.spec.NestedForms[current.Head.String()]; !ok {
			s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s cannot contain nested form %q", state.spec.Name, current.Head.String()))
			return
		}
	}
	nested, hirNested := s.compileForm(current, locals)
	if nested != nil && hirNested != nil {
		state.form.Forms = append(state.form.Forms, *nested)
		state.hir.Forms = append(state.hir.Forms, *hirNested)
	}
}

func (s *compileState) execCall(state *formExecState, current *ast.CallStmt, locals *env) {
	if !allowsCall(state.spec.BodyMode) {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow call statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	call, spec, ok := s.buildActionCall(current, locals)
	if !ok {
		return
	}
	state.form.Calls = append(state.form.Calls, call)
	state.hir.Calls = append(state.hir.Calls, s.lowerActionCall(current, locals.scope, call, spec))
}

func (s *compileState) execLocalBinding(state *formExecState, kind LocalBindingKind, name string, typeExpr ast.TypeExpr, expr ast.Expr, locals *env) {
	if state.spec.BodyMode != schema.BodyScript {
		s.diags.AddError(expr.Pos(), expr.End(), fmt.Sprintf("%s does not allow script statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	if _, ok := state.spec.Fields[name]; ok {
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

func (s *compileState) execFormLoopControl(state *formExecState, pos, end token.Pos, isBreak bool) execSignal {
	if state.spec.BodyMode != schema.BodyScript {
		s.diags.AddError(pos, end, fmt.Sprintf("%s does not allow script statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return noExecSignal()
	}
	if isBreak {
		return breakExecSignal(pos, end)
	}
	return continueExecSignal(pos, end)
}

func (s *compileState) execIf(state *formExecState, current *ast.IfStmt, locals *env) execSignal {
	if state.spec.BodyMode != schema.BodyScript {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow script statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return noExecSignal()
	}
	value, err := s.evalExpr(current.Condition, locals)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return noExecSignal()
	}
	cond, ok := value.(bool)
	if !ok {
		s.diags.AddError(current.Pos(), current.End(), "if condition must be bool")
		return noExecSignal()
	}
	if cond {
		return s.execFormBlock(state, current.Then, locals)
	}
	if current.Else != nil {
		return s.execFormBlock(state, current.Else, locals)
	}
	return noExecSignal()
}

func (s *compileState) execFor(state *formExecState, current *ast.ForStmt, locals *env) execSignal {
	if state.spec.BodyMode != schema.BodyScript {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow script statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return noExecSignal()
	}
	value, err := s.evalExpr(current.Iterable, locals)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return noExecSignal()
	}
	items, err := iterateValues(value)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return noExecSignal()
	}
	if _, ok := state.spec.Fields[current.Name.Name]; ok {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("loop variable %q conflicts with field %q in %s", current.Name.Name, current.Name.Name, state.spec.Name))
		return noExecSignal()
	}
	for _, item := range items {
		blockEnv := s.newScopeEnv(locals, ScopeLoop, current.Pos(), current.End())
		blockEnv.BindLocal(current.Name.Name, LocalLoop, staticTypeOfValue(item), item)
		signal := s.execFormBlock(state, current.Body, blockEnv)
		switch {
		case signal.IsBreak():
			return noExecSignal()
		case signal.IsContinue():
			continue
		case !signal.IsNone():
			return signal
		}
	}
	return noExecSignal()
}

func (s *compileState) execFormBlock(state *formExecState, block *ast.Block, locals *env) execSignal {
	if block == nil {
		return noExecSignal()
	}
	return s.execFormItems(state, block.Items, s.newScopeEnv(locals, ScopeBlock, block.Pos(), block.End()))
}
