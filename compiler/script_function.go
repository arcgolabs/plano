package compiler

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/mo"
)

func (s *compileState) callUserFunction(name string, decl *ast.FnDecl, args []any) (any, error) {
	if len(args) != len(decl.Params) {
		return nil, fmt.Errorf("function %q expects %d arguments", name, len(decl.Params))
	}
	locals, err := s.bindFunctionParams(name, decl, args)
	if err != nil {
		return nil, err
	}
	result, err := s.execFunctionBlock(decl.Body, locals)
	if err != nil {
		return nil, fmt.Errorf("function %q: %w", name, err)
	}
	if err := unexpectedLoopControlError(result); err != nil {
		return nil, fmt.Errorf("function %q: %w", name, err)
	}
	if err := validateFunctionResult(name, decl.Result, result.Result()); err != nil {
		return nil, err
	}
	if value, ok := result.Result().Get(); ok {
		return value, nil
	}
	return schema.Null{}, nil
}

func (s *compileState) bindFunctionParams(name string, decl *ast.FnDecl, args []any) (*env, error) {
	locals := s.newScopeEnv(nil, ScopeFunction, decl.Pos(), decl.End())
	for idx, param := range decl.Params {
		value := args[idx]
		if err := validateFunctionParam(name, param, value); err != nil {
			return nil, err
		}
		locals.BindLocal(param.Name.Name, LocalParam, convertTypeExpr(param.Type), value)
	}
	return locals, nil
}

func validateFunctionParam(name string, param *ast.Param, value any) error {
	if param.Type == nil {
		return nil
	}
	typ := convertTypeExpr(param.Type)
	if typ == nil {
		return nil
	}
	if err := schema.CheckAssignable(typ, value); err != nil {
		return fmt.Errorf("function %q parameter %q: %w", name, param.Name.Name, err)
	}
	return nil
}

func validateFunctionResult(name string, typeExpr ast.TypeExpr, result mo.Option[any]) error {
	if typeExpr == nil {
		return nil
	}
	if result.IsAbsent() {
		return fmt.Errorf("function %q must return a value", name)
	}
	typ := convertTypeExpr(typeExpr)
	if typ == nil {
		return nil
	}
	if err := schema.CheckAssignable(typ, result.MustGet()); err != nil {
		return fmt.Errorf("function %q return type: %w", name, err)
	}
	return nil
}

func (s *compileState) execFunctionBlock(block *ast.Block, locals *env) (execSignal, error) {
	if block == nil {
		return noExecSignal(), nil
	}
	for _, item := range block.Items {
		result, handled, err := s.execFunctionItem(item, locals)
		if !handled || err != nil || !result.IsNone() {
			return result, err
		}
	}
	return noExecSignal(), nil
}

func (s *compileState) execFunctionItem(item ast.FormItem, locals *env) (execSignal, bool, error) {
	if signal, handled, err := s.execFunctionBindingItem(item, locals); handled {
		return signal, true, err
	}
	if signal, handled, err := s.execFunctionControlItem(item, locals); handled {
		return signal, true, err
	}
	return s.execFunctionTerminalItem(item, locals)
}

func (s *compileState) execFunctionBindingItem(item ast.FormItem, locals *env) (execSignal, bool, error) {
	switch current := item.(type) {
	case *ast.ConstDecl:
		return noExecSignal(), true, s.bindFunctionLocal(LocalConst, current.Name.Name, current.Type, current.Value, locals)
	case *ast.LetDecl:
		return noExecSignal(), true, s.bindFunctionLocal(LocalLet, current.Name.Name, current.Type, current.Value, locals)
	case *ast.Assignment:
		return noExecSignal(), true, s.execFunctionAssignment(current, locals)
	default:
		return noExecSignal(), false, nil
	}
}

func (s *compileState) execFunctionControlItem(item ast.FormItem, locals *env) (execSignal, bool, error) {
	switch current := item.(type) {
	case *ast.IfStmt:
		return s.execFunctionIf(current, locals)
	case *ast.ForStmt:
		return s.execFunctionFor(current, locals)
	default:
		return noExecSignal(), false, nil
	}
}

func (s *compileState) execFunctionTerminalItem(item ast.FormItem, locals *env) (execSignal, bool, error) {
	switch current := item.(type) {
	case *ast.ReturnStmt:
		value, err := s.evalExpr(current.Value, locals)
		if err != nil {
			return noExecSignal(), true, err
		}
		return returnExecSignal(current.Pos(), current.End(), value), true, nil
	case *ast.BreakStmt:
		return breakExecSignal(current.Pos(), current.End()), true, nil
	case *ast.ContinueStmt:
		return continueExecSignal(current.Pos(), current.End()), true, nil
	default:
		return noExecSignal(), true, unsupportedFunctionItemError(current)
	}
}

func (s *compileState) execFunctionIf(stmt *ast.IfStmt, locals *env) (execSignal, bool, error) {
	value, err := s.evalExpr(stmt.Condition, locals)
	if err != nil {
		return noExecSignal(), true, err
	}
	cond, ok := value.(bool)
	if !ok {
		return noExecSignal(), true, errors.New("if condition must be bool")
	}
	branch := stmt.Else
	if cond {
		branch = stmt.Then
	}
	if branch == nil {
		return noExecSignal(), true, nil
	}
	result, err := s.execFunctionBlock(branch, s.newScopeEnv(locals, ScopeBlock, branch.Pos(), branch.End()))
	return result, true, err
}

func (s *compileState) execFunctionFor(stmt *ast.ForStmt, locals *env) (execSignal, bool, error) {
	items, err := s.evalFunctionLoopItems(stmt, locals)
	if err != nil {
		return noExecSignal(), true, err
	}
	for _, item := range items {
		result, cont, err := s.execFunctionLoopIteration(stmt, locals, item)
		if err != nil {
			return result, true, err
		}
		if next, done := normalizeFunctionLoopResult(result, cont); done {
			return next, true, nil
		}
		if cont {
			continue
		}
	}
	return noExecSignal(), true, nil
}

func normalizeFunctionLoopResult(result execSignal, cont bool) (execSignal, bool) {
	switch {
	case result.IsBreak():
		return noExecSignal(), true
	case cont || result.IsContinue():
		return noExecSignal(), false
	case !result.IsNone():
		return result, true
	default:
		return noExecSignal(), false
	}
}

func (s *compileState) evalFunctionLoopItems(stmt *ast.ForStmt, locals *env) ([]iterItem, error) {
	value, err := s.evalExpr(stmt.Iterable, locals)
	if err != nil {
		return nil, err
	}
	return iterateItems(value)
}

func (s *compileState) execFunctionLoopIteration(stmt *ast.ForStmt, locals *env, item iterItem) (execSignal, bool, error) {
	blockEnv := s.newScopeEnv(locals, ScopeLoop, stmt.Pos(), stmt.End())
	if stmt.Index != nil {
		blockEnv.BindLocal(stmt.Index.Name, LocalLoop, staticTypeOfValue(item.Key), item.Key)
	}
	blockEnv.BindLocal(stmt.Name.Name, LocalLoop, staticTypeOfValue(item.Value), item.Value)
	run, err := s.evalFunctionLoopFilter(stmt, blockEnv)
	if err != nil || !run {
		return noExecSignal(), !run, err
	}
	result, err := s.execFunctionBlock(stmt.Body, blockEnv)
	return result, false, err
}

func (s *compileState) evalFunctionLoopFilter(stmt *ast.ForStmt, locals *env) (bool, error) {
	if stmt.Filter == nil {
		return true, nil
	}
	value, err := s.evalExpr(stmt.Filter, locals)
	if err != nil {
		return false, err
	}
	run, ok := value.(bool)
	if !ok {
		return false, errors.New("for where clause must be bool")
	}
	return run, nil
}

func unsupportedFunctionItemError(item ast.FormItem) error {
	switch current := item.(type) {
	case *ast.ImportDecl:
		return errors.New("import is not allowed in function bodies")
	case *ast.FnDecl:
		return errors.New("nested function declarations are not implemented")
	case *ast.Assignment, *ast.CallStmt, *ast.FormDecl:
		return fmt.Errorf("unsupported function body item %T", current)
	default:
		return fmt.Errorf("unsupported function body item %T", current)
	}
}

func (s *compileState) execFunctionAssignment(assign *ast.Assignment, locals *env) error {
	value, err := s.evalExpr(assign.Value, locals)
	if err != nil {
		return err
	}
	return locals.Assign(assign.Name.Name, value)
}

func (s *compileState) bindFunctionLocal(kind LocalBindingKind, name string, typeExpr ast.TypeExpr, expr ast.Expr, locals *env) error {
	value, err := s.evalExpr(expr, locals)
	if err != nil {
		return err
	}
	return s.bindLocalValue(locals, kind, name, typeExpr, value)
}
