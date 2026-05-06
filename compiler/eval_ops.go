package compiler

import (
	"reflect"

	"github.com/arcgolabs/collectionx/mapping"
)

func evalUnary(op string, value any) (any, error) {
	switch op {
	case "!":
		v, ok := value.(bool)
		if !ok {
			return nil, compilerErrorf("operator ! expects bool")
		}
		return !v, nil
	case "-":
		return negateNumber(value)
	default:
		return nil, compilerErrorf("unsupported unary operator %q", op)
	}
}

func negateNumber(value any) (any, error) {
	switch v := value.(type) {
	case int64:
		return -v, nil
	case float64:
		return -v, nil
	default:
		return nil, compilerErrorf("operator - expects number")
	}
}

func evalBinary(op string, left, right any) (any, error) {
	switch {
	case op == "+":
		if value, ok, err := concatStrings(left, right); ok {
			return value, err
		}
		return numericBinary(op, left, right)
	case isNumericOperator(op):
		return numericBinary(op, left, right)
	case op == "%":
		return moduloInts(left, right)
	case isEqualityOp(op):
		return evalEqualityBinary(op, left, right), nil
	default:
		return evalPredicateBinary(op, left, right)
	}
}

func evalEqualityBinary(op string, left, right any) bool {
	if op == "==" {
		return reflect.DeepEqual(left, right)
	}
	return !reflect.DeepEqual(left, right)
}

func evalPredicateBinary(op string, left, right any) (any, error) {
	if op == "&&" || op == "||" {
		return logicalBinary(op, left, right)
	}
	if op == "in" {
		return evalContains(right, left)
	}
	if isComparisonOperator(op) {
		return compareBinary(op, left, right)
	}
	return nil, compilerErrorf("unsupported operator %q", op)
}

func concatStrings(left, right any) (any, bool, error) {
	l, ok := left.(string)
	if !ok {
		return nil, false, nil
	}
	r, ok := right.(string)
	if !ok {
		return nil, true, compilerErrorf("operator + expects string/string or number/number")
	}
	return l + r, true, nil
}

func moduloInts(left, right any) (any, error) {
	l, lok := left.(int64)
	r, rok := right.(int64)
	if !lok || !rok {
		return nil, compilerErrorf("operator %% expects int operands")
	}
	return l % r, nil
}

func logicalBinary(op string, left, right any) (any, error) {
	l, lok := left.(bool)
	r, rok := right.(bool)
	if !lok || !rok {
		return nil, compilerErrorf("operator %s expects bool operands", op)
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
	return nil, compilerErrorf("operator %s expects numeric operands", op)
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
	return nil, compilerErrorf("unsupported numeric operator %q", op)
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
	return nil, compilerErrorf("unsupported numeric operator %q", op)
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
			return nil, compilerErrorf("comparison expects compatible operands")
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
		return nil, true, compilerErrorf("comparison expects compatible operands")
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
		return nil, true, compilerErrorf("unsupported comparison %q", op)
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
		return nil, compilerErrorf("unsupported comparison %q", op)
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
		return nil, compilerErrorf("indexing is not supported on %T", base)
	}
}

func evalSliceIndex(collection []any, index any) (any, error) {
	i, ok := index.(int64)
	if !ok {
		return nil, compilerErrorf("array index expects int")
	}
	if i < 0 || int(i) >= len(collection) {
		return nil, compilerErrorf("array index out of range")
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
		return nil, compilerErrorf("unknown key %q", key)
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
		return nil, compilerErrorf("unknown key %q", key)
	}
	return value, nil
}

func stringKey(index any) (string, error) {
	key, ok := index.(string)
	if !ok {
		return "", compilerErrorf("object index expects string")
	}
	return key, nil
}

func isNumericOperator(op string) bool {
	return op == "-" || op == "*" || op == "/"
}

func isComparisonOperator(op string) bool {
	return op == ">" || op == ">=" || op == "<" || op == "<="
}
