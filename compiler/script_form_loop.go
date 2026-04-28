package compiler

import (
	"fmt"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (s *compileState) execFor(state *formExecState, current *ast.ForStmt, locals *env) execSignal {
	if state.spec.BodyMode != schema.BodyScript {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow script statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return noExecSignal()
	}
	items, ok := s.evalLoopItems(current, locals)
	if !ok {
		return noExecSignal()
	}
	if err := validateFormLoopVars(state.spec, current); err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return noExecSignal()
	}
	for _, item := range items {
		if signal, stop := s.execFormLoopIteration(state, current, locals, item); stop {
			return signal
		}
	}
	return noExecSignal()
}

func (s *compileState) evalLoopItems(current *ast.ForStmt, locals *env) ([]iterItem, bool) {
	value, err := s.evalExpr(current.Iterable, locals)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return nil, false
	}
	items, err := iterateItems(value)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return nil, false
	}
	return items, true
}

func (s *compileState) execFormLoopIteration(state *formExecState, current *ast.ForStmt, locals *env, item iterItem) (execSignal, bool) {
	blockEnv := s.newScopeEnv(locals, ScopeLoop, current.Pos(), current.End())
	if current.Index != nil {
		blockEnv.BindLocal(current.Index.Name, LocalLoop, staticTypeOfValue(item.Key), item.Key)
	}
	blockEnv.BindLocal(current.Name.Name, LocalLoop, staticTypeOfValue(item.Value), item.Value)
	signal := s.execFormBlock(state, current.Body, blockEnv)
	switch {
	case signal.IsBreak():
		return noExecSignal(), true
	case signal.IsContinue():
		return noExecSignal(), false
	case !signal.IsNone():
		return signal, true
	default:
		return noExecSignal(), false
	}
}

func validateFormLoopVars(spec schema.FormSpec, stmt *ast.ForStmt) error {
	if stmt.Index != nil {
		if _, ok := spec.Fields[stmt.Index.Name]; ok {
			return fmt.Errorf("loop variable %q conflicts with field %q in %s", stmt.Index.Name, stmt.Index.Name, spec.Name)
		}
	}
	if _, ok := spec.Fields[stmt.Name.Name]; ok {
		return fmt.Errorf("loop variable %q conflicts with field %q in %s", stmt.Name.Name, stmt.Name.Name, spec.Name)
	}
	return nil
}
