//nolint:cyclop,gocognit,gocyclo // Expression evaluation centralizes operator semantics in one file.
package compiler

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (c *Compiler) registerBuiltins() {
	c.mustRegisterFunc(builtinFunction("env", 2, func(args []any) (any, error) {
		key, ok := args[0].(string)
		if !ok {
			return nil, errors.New("env expects string key")
		}
		if value, ok := c.lookupEnv(key); ok {
			return value, nil
		}
		if len(args) == 2 {
			if fallback, ok := args[1].(string); ok {
				return fallback, nil
			}
			return nil, errors.New("env fallback expects string")
		}
		return "", nil
	}))
	c.mustRegisterFunc(builtinFunction("join_path", -1, func(args []any) (any, error) {
		parts := make([]string, 0, len(args))
		for _, arg := range args {
			text, ok := arg.(string)
			if !ok {
				return nil, errors.New("join_path expects string arguments")
			}
			parts = append(parts, text)
		}
		return filepath.Join(parts...), nil
	}))
	c.mustRegisterFunc(builtinFunction("basename", 1, func(args []any) (any, error) {
		path, ok := args[0].(string)
		if !ok {
			return nil, errors.New("basename expects string argument")
		}
		return filepath.Base(path), nil
	}))
	c.mustRegisterFunc(builtinFunction("dirname", 1, func(args []any) (any, error) {
		path, ok := args[0].(string)
		if !ok {
			return nil, errors.New("dirname expects string argument")
		}
		return filepath.Dir(path), nil
	}))
}

func (c *Compiler) mustRegisterFunc(spec schema.FunctionSpec) {
	if err := c.RegisterFunc(spec); err != nil {
		panic(fmt.Errorf("register builtin function %q: %w", spec.Name, err))
	}
}

func builtinFunction(name string, maxArgs int, eval func(args []any) (any, error)) schema.FunctionSpec {
	return schema.FunctionSpec{
		Name:    name,
		MinArgs: 1,
		MaxArgs: maxArgs,
		Eval:    eval,
	}
}

func evalUnary(op string, value any) (any, error) {
	switch op {
	case "!":
		v, ok := value.(bool)
		if !ok {
			return nil, errors.New("operator ! expects bool")
		}
		return !v, nil
	case "-":
		switch v := value.(type) {
		case int64:
			return -v, nil
		case float64:
			return -v, nil
		default:
			return nil, errors.New("operator - expects number")
		}
	default:
		return nil, fmt.Errorf("unsupported unary operator %q", op)
	}
}

func evalBinary(op string, left, right any) (any, error) {
	switch op {
	case "+":
		if l, ok := left.(string); ok {
			r, ok := right.(string)
			if !ok {
				return nil, errors.New("operator + expects string/string or number/number")
			}
			return l + r, nil
		}
		return numericBinary(op, left, right)
	case "-", "*", "/":
		return numericBinary(op, left, right)
	case "%":
		l, lok := left.(int64)
		r, rok := right.(int64)
		if !lok || !rok {
			return nil, errors.New("operator % expects int operands")
		}
		return l % r, nil
	case "==":
		return reflect.DeepEqual(left, right), nil
	case "!=":
		return !reflect.DeepEqual(left, right), nil
	case "&&":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, errors.New("operator && expects bool operands")
		}
		return l && r, nil
	case "||":
		l, lok := left.(bool)
		r, rok := right.(bool)
		if !lok || !rok {
			return nil, errors.New("operator || expects bool operands")
		}
		return l || r, nil
	case ">", ">=", "<", "<=":
		return compareBinary(op, left, right)
	default:
		return nil, fmt.Errorf("unsupported operator %q", op)
	}
}

func numericBinary(op string, left, right any) (any, error) {
	if lf, rf, ok := asFloatPair(left, right); ok {
		switch op {
		case "+":
			return lf + rf, nil
		case "-":
			return lf - rf, nil
		case "*":
			return lf * rf, nil
		case "/":
			return lf / rf, nil
		}
	}
	li, lok := left.(int64)
	ri, rok := right.(int64)
	if lok && rok {
		switch op {
		case "+":
			return li + ri, nil
		case "-":
			return li - ri, nil
		case "*":
			return li * ri, nil
		case "/":
			return li / ri, nil
		}
	}
	return nil, fmt.Errorf("operator %s expects numeric operands", op)
}

func compareBinary(op string, left, right any) (any, error) {
	if ls, ok := left.(string); ok {
		rs, ok := right.(string)
		if !ok {
			return nil, errors.New("comparison expects compatible operands")
		}
		switch op {
		case ">":
			return ls > rs, nil
		case ">=":
			return ls >= rs, nil
		case "<":
			return ls < rs, nil
		case "<=":
			return ls <= rs, nil
		}
	}
	lf, rf, ok := asFloatPair(left, right)
	if !ok {
		li, lok := left.(int64)
		ri, rok := right.(int64)
		if !lok || !rok {
			return nil, errors.New("comparison expects compatible operands")
		}
		lf = float64(li)
		rf = float64(ri)
	}
	switch op {
	case ">":
		return lf > rf, nil
	case ">=":
		return lf >= rf, nil
	case "<":
		return lf < rf, nil
	case "<=":
		return lf <= rf, nil
	default:
		return nil, fmt.Errorf("unsupported comparison %q", op)
	}
}

func asFloatPair(left, right any) (float64, float64, bool) {
	var lf, rf float64
	switch l := left.(type) {
	case float64:
		lf = l
	case int64:
		lf = float64(l)
	default:
		return 0, 0, false
	}
	switch r := right.(type) {
	case float64:
		rf = r
	case int64:
		rf = float64(r)
	default:
		return 0, 0, false
	}
	return lf, rf, true
}

func evalIndex(base, index any) (any, error) {
	switch collection := base.(type) {
	case []any:
		i, ok := index.(int64)
		if !ok {
			return nil, errors.New("array index expects int")
		}
		if i < 0 || int(i) >= len(collection) {
			return nil, errors.New("array index out of range")
		}
		return collection[i], nil
	case map[string]any:
		key, ok := index.(string)
		if !ok {
			return nil, errors.New("object index expects string")
		}
		value, ok := collection[key]
		if !ok {
			return nil, fmt.Errorf("unknown key %q", key)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("indexing is not supported on %T", base)
	}
}

func callName(expr ast.Expr) (string, bool) {
	switch node := expr.(type) {
	case *ast.IdentExpr:
		return node.Name.Name, true
	case *ast.SelectorExpr:
		prefix, ok := callName(node.X)
		if !ok {
			return "", false
		}
		return prefix + "." + node.Sel.Name, true
	default:
		return "", false
	}
}
