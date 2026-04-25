// Package schema defines host-registered forms, fields, and scalar types for plano.
//
//nolint:cyclop,gocognit,gocyclo // Type checking is centralized here to keep host schema rules together.
package schema

import (
	"fmt"
	"strconv"
	"strings"
	"time"
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
	Fields       map[string]FieldSpec
	NestedForms  map[string]struct{}
	Declares     string
	Docs         string
}

type FunctionSpec struct {
	Name    string
	MinArgs int
	MaxArgs int
	Eval    func(args []any) (any, error)
	Docs    string
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
	case MapType:
		items, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("expected %s, got %T", want.String(), value)
		}
		for _, item := range items {
			if err := CheckAssignable(want.Elem, item); err != nil {
				return err
			}
		}
		return nil
	case RefType:
		ref, ok := value.(Ref)
		if !ok {
			return fmt.Errorf("expected %s, got %T", want.String(), value)
		}
		if ref.Kind != want.Kind {
			return fmt.Errorf("expected %s, got ref<%s>", want.String(), ref.Kind)
		}
		return nil
	case NamedType:
		return nil
	default:
		return fmt.Errorf("unsupported type %T", t)
	}
}

func checkBuiltin(want BuiltinType, value any) error {
	switch want {
	case TypeAny:
		return nil
	case TypeString, TypePath:
		if _, ok := value.(string); ok {
			return nil
		}
	case TypeInt:
		if _, ok := value.(int64); ok {
			return nil
		}
	case TypeFloat:
		if _, ok := value.(float64); ok {
			return nil
		}
	case TypeBool:
		if _, ok := value.(bool); ok {
			return nil
		}
	case TypeDuration:
		if _, ok := value.(Duration); ok {
			return nil
		}
	case TypeSize:
		if _, ok := value.(Size); ok {
			return nil
		}
	}
	return fmt.Errorf("expected %s, got %T", want.String(), value)
}
