package compiler

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (s *compileState) evalExpr(expr ast.Expr, locals *env) (any, error) {
	if value, handled, err := s.evalLiteralExpr(expr); handled {
		return value, err
	}
	if value, handled, err := s.evalDirectExpr(expr, locals); handled {
		return value, err
	}
	return s.evalCompoundExpr(expr, locals)
}

func (s *compileState) evalLiteralExpr(expr ast.Expr) (any, bool, error) {
	if value, ok := evalScalarLiteral(expr); ok {
		return value, true, nil
	}
	switch node := expr.(type) {
	case *ast.NullLiteral:
		return schema.Null{}, true, nil
	case *ast.DurationLiteral:
		return parseMeasuredLiteral(node.Raw, schema.ParseDuration, "duration")
	case *ast.SizeLiteral:
		return parseMeasuredLiteral(node.Raw, schema.ParseSize, "size")
	default:
		return nil, false, nil
	}
}

func evalScalarLiteral(expr ast.Expr) (any, bool) {
	switch node := expr.(type) {
	case *ast.StringLiteral:
		return node.Value, true
	case *ast.IntLiteral:
		return node.Value, true
	case *ast.FloatLiteral:
		return node.Value, true
	case *ast.BoolLiteral:
		return node.Value, true
	default:
		return nil, false
	}
}

func parseMeasuredLiteral[T any](raw string, parse func(string) (T, error), label string) (any, bool, error) {
	value, err := parse(raw)
	if err != nil {
		return nil, true, fmt.Errorf("parse %s %q: %w", label, raw, err)
	}
	return value, true, nil
}

func (s *compileState) evalDirectExpr(expr ast.Expr, locals *env) (any, bool, error) {
	switch node := expr.(type) {
	case *ast.IdentExpr:
		value, err := s.resolveName(node.Name.Name, locals)
		return value, true, err
	case *ast.ArrayExpr:
		value, err := s.evalArrayExpr(node, locals)
		return value, true, err
	case *ast.ObjectExpr:
		value, err := s.evalObjectExpr(node, locals)
		return value, true, err
	case *ast.ParenExpr:
		value, err := s.evalExpr(node.X, locals)
		return value, true, err
	default:
		return nil, false, nil
	}
}

func (s *compileState) evalCompoundExpr(expr ast.Expr, locals *env) (any, error) {
	switch node := expr.(type) {
	case *ast.UnaryExpr:
		value, err := s.evalExpr(node.X, locals)
		if err != nil {
			return nil, err
		}
		return evalUnary(node.Op, value)
	case *ast.BinaryExpr:
		return s.evalBinaryExpr(node, locals)
	case *ast.SelectorExpr:
		return s.evalSelectorExpr(node, locals)
	case *ast.IndexExpr:
		base, index, err := s.evalIndexParts(node, locals)
		if err != nil {
			return nil, err
		}
		return evalIndex(base, index)
	case *ast.CallExpr:
		return s.evalCallExpr(node, locals)
	default:
		return nil, fmt.Errorf("unsupported expression %T", expr)
	}
}

func (s *compileState) evalArrayExpr(node *ast.ArrayExpr, locals *env) ([]any, error) {
	items := make([]any, 0, len(node.Elements))
	for _, elem := range node.Elements {
		value, err := s.evalExpr(elem, locals)
		if err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	return items, nil
}

func (s *compileState) evalObjectExpr(node *ast.ObjectExpr, locals *env) (*mapping.OrderedMap[string, any], error) {
	items := mapping.NewOrderedMapWithCapacity[string, any](len(node.Entries))
	for _, entry := range node.Entries {
		value, err := s.evalExpr(entry.Value, locals)
		if err != nil {
			return nil, err
		}
		items.Set(entry.Key.Name, value)
	}
	return items, nil
}

func (s *compileState) evalBinaryExpr(node *ast.BinaryExpr, locals *env) (any, error) {
	left, err := s.evalExpr(node.X, locals)
	if err != nil {
		return nil, err
	}
	right, err := s.evalExpr(node.Y, locals)
	if err != nil {
		return nil, err
	}
	return evalBinary(node.Op, left, right)
}

func (s *compileState) evalSelectorExpr(node *ast.SelectorExpr, locals *env) (any, error) {
	base, err := s.evalExpr(node.X, locals)
	if err != nil {
		return nil, err
	}
	switch object := base.(type) {
	case *mapping.OrderedMap[string, any]:
		value, ok := object.Get(node.Sel.Name)
		if !ok {
			return nil, fmt.Errorf("unknown field %q", node.Sel.Name)
		}
		return value, nil
	case map[string]any:
		value, ok := object[node.Sel.Name]
		if !ok {
			return nil, fmt.Errorf("unknown field %q", node.Sel.Name)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("selector requires object, got %T", base)
	}
}

func (s *compileState) evalIndexParts(node *ast.IndexExpr, locals *env) (any, any, error) {
	base, err := s.evalExpr(node.X, locals)
	if err != nil {
		return nil, nil, err
	}
	index, err := s.evalExpr(node.Index, locals)
	if err != nil {
		return nil, nil, err
	}
	return base, index, nil
}

func (s *compileState) evalCallExpr(node *ast.CallExpr, locals *env) (any, error) {
	name, ok := callName(node.Fun)
	if !ok {
		return nil, errors.New("unsupported call target")
	}
	args, err := s.evalCallArgs(node.Args, locals)
	if err != nil {
		return nil, err
	}
	if spec, ok := s.compiler.funcs.Get(name); ok {
		if err := validateArity("function", name, spec.MinArgs, spec.MaxArgs, len(args)); err != nil {
			return nil, err
		}
		value, err := spec.Eval(args)
		if err != nil {
			return nil, fmt.Errorf("evaluate function %q: %w", name, err)
		}
		return value, nil
	}
	if decl, ok := s.funcDecls.Get(name); ok {
		return s.callUserFunction(name, decl, args)
	}
	return nil, fmt.Errorf("unknown function %q", name)
}

func (s *compileState) evalCallArgs(args []ast.Expr, locals *env) ([]any, error) {
	values := make([]any, 0, len(args))
	for _, arg := range args {
		value, err := s.evalExpr(arg, locals)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (s *compileState) resolveName(name string, locals *env) (any, error) {
	if locals != nil {
		if value, ok := locals.Get(name); ok {
			return value, nil
		}
	}
	if value, ok := s.compiler.globals.Get(name); ok {
		return value, nil
	}
	if value, ok := s.constValues.Get(name); ok {
		return value, nil
	}
	if _, ok := s.constDecls.Get(name); ok {
		value, ok := s.resolveConst(name)
		if ok {
			return value, nil
		}
		return nil, fmt.Errorf("failed to resolve constant %q", name)
	}
	if symbol, ok := s.symbols.Get(name); ok {
		return schema.Ref{Kind: symbol.Kind, Name: symbol.Name}, nil
	}
	return nil, fmt.Errorf("undefined symbol %q", name)
}
