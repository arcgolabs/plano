package compiler

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/collectionx/mapping"
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
	if err := validateFunctionResult(name, decl.Result, result); err != nil {
		return nil, err
	}
	if value, ok := result.Get(); ok {
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
		locals.Bind(param.Name.Name, value)
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

func (s *compileState) execFunctionBlock(block *ast.Block, locals *env) (mo.Option[any], error) {
	if block == nil {
		return mo.None[any](), nil
	}
	for _, item := range block.Items {
		result, handled, err := s.execFunctionItem(item, locals)
		if !handled || err != nil || result.IsPresent() {
			return result, err
		}
	}
	return mo.None[any](), nil
}

func (s *compileState) execFunctionItem(item ast.FormItem, locals *env) (mo.Option[any], bool, error) {
	switch current := item.(type) {
	case *ast.ConstDecl:
		return mo.None[any](), true, s.bindFunctionLocal(current.Name.Name, current.Type, current.Value, locals)
	case *ast.LetDecl:
		return mo.None[any](), true, s.bindFunctionLocal(current.Name.Name, current.Type, current.Value, locals)
	case *ast.IfStmt:
		return s.execFunctionIf(current, locals)
	case *ast.ForStmt:
		return s.execFunctionFor(current, locals)
	case *ast.ReturnStmt:
		value, err := s.evalExpr(current.Value, locals)
		if err != nil {
			return mo.None[any](), true, err
		}
		return mo.Some[any](value), true, nil
	default:
		return mo.None[any](), true, unsupportedFunctionItemError(current)
	}
}

func (s *compileState) execFunctionIf(stmt *ast.IfStmt, locals *env) (mo.Option[any], bool, error) {
	value, err := s.evalExpr(stmt.Condition, locals)
	if err != nil {
		return mo.None[any](), true, err
	}
	cond, ok := value.(bool)
	if !ok {
		return mo.None[any](), true, errors.New("if condition must be bool")
	}
	branch := stmt.Else
	if cond {
		branch = stmt.Then
	}
	if branch == nil {
		return mo.None[any](), true, nil
	}
	result, err := s.execFunctionBlock(branch, s.newScopeEnv(locals, ScopeBlock, branch.Pos(), branch.End()))
	return result, true, err
}

func (s *compileState) execFunctionFor(stmt *ast.ForStmt, locals *env) (mo.Option[any], bool, error) {
	value, err := s.evalExpr(stmt.Iterable, locals)
	if err != nil {
		return mo.None[any](), true, err
	}
	items, err := iterateValues(value)
	if err != nil {
		return mo.None[any](), true, err
	}
	for _, itemValue := range items {
		blockEnv := s.newScopeEnv(locals, ScopeLoop, stmt.Pos(), stmt.End())
		blockEnv.Bind(stmt.Name.Name, itemValue)
		result, err := s.execFunctionBlock(stmt.Body, blockEnv)
		if err != nil || result.IsPresent() {
			return result, true, err
		}
	}
	return mo.None[any](), true, nil
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

func (s *compileState) bindFunctionLocal(name string, typeExpr ast.TypeExpr, expr ast.Expr, locals *env) error {
	value, err := s.evalExpr(expr, locals)
	if err != nil {
		return err
	}
	if typeExpr != nil {
		if typ := convertTypeExpr(typeExpr); typ != nil {
			if err := schema.CheckAssignable(typ, value); err != nil {
				return fmt.Errorf("binding %q: %w", name, err)
			}
		}
	}
	locals.Bind(name, value)
	return nil
}

func iterateValues(value any) ([]any, error) {
	switch current := value.(type) {
	case []any:
		return current, nil
	case *mapping.OrderedMap[string, any]:
		return current.Values(), nil
	case map[string]any:
		items := make([]any, 0, len(current))
		for _, item := range current {
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("for loop expects list or map, got %T", value)
	}
}
