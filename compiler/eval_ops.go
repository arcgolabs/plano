package compiler

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/arcgolabs/collectionx/mapping"
)

func evalUnary(op string, value any) (any, error) {
	switch op {
	case "!":
		v, ok := value.(bool)
		if !ok {
			return nil, errors.New("operator ! expects bool")
		}
		return !v, nil
	case "-":
		return negateNumber(value)
	default:
		return nil, fmt.Errorf("unsupported unary operator %q", op)
	}
}

func negateNumber(value any) (any, error) {
	switch v := value.(type) {
	case int64:
		return -v, nil
	case float64:
		return -v, nil
	default:
		return nil, errors.New("operator - expects number")
	}
}

func evalBinary(op string, left, right any) (any, error) {
	if op == "+" {
		if value, ok, err := concatStrings(left, right); ok {
			return value, err
		}
		return numericBinary(op, left, right)
	}
	if isNumericOperator(op) {
		return numericBinary(op, left, right)
	}
	if op == "%" {
		return moduloInts(left, right)
	}
	if op == "==" {
		return reflect.DeepEqual(left, right), nil
	}
	if op == "!=" {
		return !reflect.DeepEqual(left, right), nil
	}
	if op == "&&" || op == "||" {
		return logicalBinary(op, left, right)
	}
	if isComparisonOperator(op) {
		return compareBinary(op, left, right)
	}
	return nil, fmt.Errorf("unsupported operator %q", op)
}

func concatStrings(left, right any) (any, bool, error) {
	l, ok := left.(string)
	if !ok {
		return nil, false, nil
	}
	r, ok := right.(string)
	if !ok {
		return nil, true, errors.New("operator + expects string/string or number/number")
	}
	return l + r, true, nil
}

func moduloInts(left, right any) (any, error) {
	l, lok := left.(int64)
	r, rok := right.(int64)
	if !lok || !rok {
		return nil, errors.New("operator % expects int operands")
	}
	return l % r, nil
}

func logicalBinary(op string, left, right any) (any, error) {
	l, lok := left.(bool)
	r, rok := right.(bool)
	if !lok || !rok {
		return nil, fmt.Errorf("operator %s expects bool operands", op)
	}
	if op == "&&" {
		return l && r, nil
	}
	return l || r, nil
}

func numericBinary(op string, left, right any) (any, error) {
	if lf, rf, ok := asFloatPair(left, right); ok {
		return applyFloatOp(op, lf, rf)
	}
	li, lok := left.(int64)
	ri, rok := right.(int64)
	if lok && rok {
		return applyIntOp(op, li, ri)
	}
	return nil, fmt.Errorf("operator %s expects numeric operands", op)
}

func applyFloatOp(op string, left, right float64) (any, error) {
	if op == "+" {
		return left + right, nil
	}
	if op == "-" {
		return left - right, nil
	}
	if op == "*" {
		return left * right, nil
	}
	if op == "/" {
		return left / right, nil
	}
	return nil, fmt.Errorf("unsupported numeric operator %q", op)
}

func applyIntOp(op string, left, right int64) (any, error) {
	if op == "+" {
		return left + right, nil
	}
	if op == "-" {
		return left - right, nil
	}
	if op == "*" {
		return left * right, nil
	}
	if op == "/" {
		return left / right, nil
	}
	return nil, fmt.Errorf("unsupported numeric operator %q", op)
}

func compareBinary(op string, left, right any) (any, error) {
	if value, ok, err := compareStrings(op, left, right); ok {
		return value, err
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
	return compareFloats(op, lf, rf)
}

func compareStrings(op string, left, right any) (any, bool, error) {
	ls, ok := left.(string)
	if !ok {
		return nil, false, nil
	}
	rs, ok := right.(string)
	if !ok {
		return nil, true, errors.New("comparison expects compatible operands")
	}
	switch op {
	case ">":
		return ls > rs, true, nil
	case ">=":
		return ls >= rs, true, nil
	case "<":
		return ls < rs, true, nil
	case "<=":
		return ls <= rs, true, nil
	default:
		return nil, true, fmt.Errorf("unsupported comparison %q", op)
	}
}

func compareFloats(op string, left, right float64) (any, error) {
	switch op {
	case ">":
		return left > right, nil
	case ">=":
		return left >= right, nil
	case "<":
		return left < right, nil
	case "<=":
		return left <= right, nil
	default:
		return nil, fmt.Errorf("unsupported comparison %q", op)
	}
}

func asFloatPair(left, right any) (float64, float64, bool) {
	lf, ok := numericValue(left)
	if !ok {
		return 0, 0, false
	}
	rf, ok := numericValue(right)
	if !ok {
		return 0, 0, false
	}
	return lf, rf, true
}

func numericValue(value any) (float64, bool) {
	switch current := value.(type) {
	case float64:
		return current, true
	case int64:
		return float64(current), true
	default:
		return 0, false
	}
}

func evalIndex(base, index any) (any, error) {
	switch collection := base.(type) {
	case []any:
		return evalSliceIndex(collection, index)
	case *mapping.OrderedMap[string, any]:
		return evalOrderedMapIndex(collection, index)
	case map[string]any:
		return evalBuiltinMapIndex(collection, index)
	default:
		return nil, fmt.Errorf("indexing is not supported on %T", base)
	}
}

func evalSliceIndex(collection []any, index any) (any, error) {
	i, ok := index.(int64)
	if !ok {
		return nil, errors.New("array index expects int")
	}
	if i < 0 || int(i) >= len(collection) {
		return nil, errors.New("array index out of range")
	}
	return collection[i], nil
}

func evalOrderedMapIndex(collection *mapping.OrderedMap[string, any], index any) (any, error) {
	key, err := stringKey(index)
	if err != nil {
		return nil, err
	}
	value, ok := collection.Get(key)
	if !ok {
		return nil, fmt.Errorf("unknown key %q", key)
	}
	return value, nil
}

func evalBuiltinMapIndex(collection map[string]any, index any) (any, error) {
	key, err := stringKey(index)
	if err != nil {
		return nil, err
	}
	value, ok := collection[key]
	if !ok {
		return nil, fmt.Errorf("unknown key %q", key)
	}
	return value, nil
}

func stringKey(index any) (string, error) {
	key, ok := index.(string)
	if !ok {
		return "", errors.New("object index expects string")
	}
	return key, nil
}

func isNumericOperator(op string) bool {
	return op == "-" || op == "*" || op == "/"
}

func isComparisonOperator(op string) bool {
	return op == ">" || op == ">=" || op == "<" || op == "<="
}
