// Package schema defines host-registered forms, fields, and scalar types for plano.
package schema

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/set"
)

type LabelKind int

const (
	LabelNone LabelKind = iota
	LabelSymbol
	LabelString
)

func (k LabelKind) String() string {
	switch k {
	case LabelNone:
		return "no-label"
	case LabelSymbol:
		return "symbol"
	case LabelString:
		return "string"
	default:
		return "unknown"
	}
}

type BodyMode int

const (
	BodyFieldOnly BodyMode = iota
	BodyFormOnly
	BodyMixed
	BodyCallOnly
	BodyScript
)

func (m BodyMode) String() string {
	switch m {
	case BodyFieldOnly:
		return "field-only"
	case BodyFormOnly:
		return "form-only"
	case BodyMixed:
		return "mixed"
	case BodyCallOnly:
		return "call-only"
	case BodyScript:
		return "script"
	default:
		return "unknown"
	}
}

type Type interface {
	String() string
}

type BuiltinType string

const (
	TypeString   BuiltinType = "string"
	TypeInt      BuiltinType = "int"
	TypeFloat    BuiltinType = "float"
	TypeBool     BuiltinType = "bool"
	TypeDuration BuiltinType = "duration"
	TypeSize     BuiltinType = "size"
	TypePath     BuiltinType = "path"
	TypeAny      BuiltinType = "any"
)

func (t BuiltinType) String() string { return string(t) }

type ListType struct {
	Elem Type
}

func (t ListType) String() string { return fmt.Sprintf("list<%s>", t.Elem.String()) }

type MapType struct {
	Elem Type
}

func (t MapType) String() string { return fmt.Sprintf("map<%s>", t.Elem.String()) }

type RefType struct {
	Kind string
}

func (t RefType) String() string { return fmt.Sprintf("ref<%s>", t.Kind) }

type NamedType struct {
	Name string
}

func (t NamedType) String() string { return t.Name }

type Ref struct {
	Kind string
	Name string
}

type Null struct{}

type Duration struct {
	Raw   string
	Value time.Duration
}

type Size struct {
	Raw   string
	Bytes int64
}

type FieldSpec struct {
	Name       string
	Type       Type
	Required   bool
	Default    any
	HasDefault bool
	Docs       string
}

type FormSpec struct {
	Name         string
	LabelKind    LabelKind
	LabelRefKind string
	BodyMode     BodyMode
	Fields       *mapping.OrderedMap[string, FieldSpec]
	NestedForms  *set.Set[string]
	Declares     string
	Docs         string
}

type FunctionSpec struct {
	Name         string
	MinArgs      int
	MaxArgs      int
	ParamTypes   list.List[Type]
	VariadicType Type
	Result       Type
	Eval         func(args list.List[any]) (any, error)
	Docs         string
}

func ParseDuration(raw string) (Duration, error) {
	value, err := time.ParseDuration(raw)
	if err != nil {
		return Duration{}, fmt.Errorf("parse duration %q: %w", raw, err)
	}
	return Duration{Raw: raw, Value: value}, nil
}

func ParseSize(raw string) (Size, error) {
	type unit struct {
		suffix string
		scale  int64
	}
	units := []unit{
		{"Ti", 1024 * 1024 * 1024 * 1024},
		{"Gi", 1024 * 1024 * 1024},
		{"Mi", 1024 * 1024},
		{"Ki", 1024},
		{"B", 1},
	}
	for _, item := range units {
		number, ok := strings.CutSuffix(raw, item.suffix)
		if ok {
			n, err := strconv.ParseInt(number, 10, 64)
			if err != nil {
				return Size{}, fmt.Errorf("parse size %q: %w", raw, err)
			}
			return Size{Raw: raw, Bytes: n * item.scale}, nil
		}
	}
	return Size{}, fmt.Errorf("invalid size literal %q", raw)
}

func CheckAssignable(t Type, value any) error {
	switch want := t.(type) {
	case nil:
		return nil
	case BuiltinType:
		return checkBuiltin(want, value)
	case ListType:
		return checkListAssignable(want, value)
	case MapType:
		return checkMapAssignable(want, value)
	case RefType:
		return checkRefAssignable(want, value)
	case NamedType:
		return nil
	default:
		return fmt.Errorf("unsupported type %T", t)
	}
}

func checkBuiltin(want BuiltinType, value any) error {
	if want == TypeAny {
		return nil
	}
	if isStringBuiltin(want) {
		if _, ok := value.(string); ok {
			return nil
		}
		return fmt.Errorf("expected %s, got %T", want.String(), value)
	}
	if matchesBuiltinValue(want, value) {
		return nil
	}
	return fmt.Errorf("expected %s, got %T", want.String(), value)
}

func checkListAssignable(want ListType, value any) error {
	items, ok := value.([]any)
	if !ok {
		return fmt.Errorf("expected %s, got %T", want.String(), value)
	}
	for _, item := range items {
		if err := CheckAssignable(want.Elem, item); err != nil {
			return err
		}
	}
	return nil
}

func checkMapAssignable(want MapType, value any) error {
	switch items := value.(type) {
	case *mapping.OrderedMap[string, any]:
		return checkMapValues(want.Elem, items.Values())
	case map[string]any:
		return checkBuiltinMapValues(want.Elem, items)
	default:
		return fmt.Errorf("expected %s, got %T", want.String(), value)
	}
}

func checkMapValues(elem Type, items []any) error {
	for _, item := range items {
		if err := CheckAssignable(elem, item); err != nil {
			return err
		}
	}
	return nil
}

func checkBuiltinMapValues(elem Type, items map[string]any) error {
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, item)
	}
	return checkMapValues(elem, values)
}

func checkRefAssignable(want RefType, value any) error {
	ref, ok := value.(Ref)
	if !ok {
		return fmt.Errorf("expected %s, got %T", want.String(), value)
	}
	if ref.Kind != want.Kind {
		return fmt.Errorf("expected %s, got ref<%s>", want.String(), ref.Kind)
	}
	return nil
}

func isStringBuiltin(want BuiltinType) bool {
	return want == TypeString || want == TypePath
}

func matchesBuiltinValue(want BuiltinType, value any) bool {
	switch want {
	case TypeString, TypePath, TypeAny:
		return false
	case TypeInt:
		_, ok := value.(int64)
		return ok
	case TypeFloat:
		_, ok := value.(float64)
		return ok
	case TypeBool:
		_, ok := value.(bool)
		return ok
	case TypeDuration:
		_, ok := value.(Duration)
		return ok
	case TypeSize:
		_, ok := value.(Size)
		return ok
	default:
		return false
	}
}
