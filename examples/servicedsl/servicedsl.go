// Package servicedsl is an example service topology DSL built on top of plano.
package servicedsl

import (
	"slices"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/lo"
)

type Stack struct {
	Name     string
	Services mapping.OrderedMap[string, Service]
}

type Service struct {
	Name      string
	Image     string
	Port      int64
	DependsOn list.List[string]
	Env       mapping.OrderedMap[string, string]
}

func Register(c *compiler.Compiler) error {
	if err := c.RegisterForms(serviceForms()); err != nil {
		return wrapServiceDSLErrorf(err, "register forms")
	}
	return nil
}

func Lower(hir *compiler.HIR) (*Stack, error) {
	stack := &Stack{}
	for idx := range hir.Forms.Len() {
		form, _ := hir.Forms.Get(idx)
		if err := applyRootForm(stack, form); err != nil {
			return nil, err
		}
	}
	if stack.Name == "" {
		return nil, serviceDSLErrorf("stack form is required")
	}
	return stack, nil
}

func applyRootForm(stack *Stack, form compiler.HIRForm) error {
	switch form.Kind {
	case "stack":
		name, err := requiredStringField(form, "name")
		if err != nil {
			return err
		}
		if stack.Name != "" {
			return serviceDSLErrorf("only one stack form is allowed")
		}
		stack.Name = name
	case "service":
		service, err := lowerService(form)
		if err != nil {
			return err
		}
		stack.Services.Set(service.Name, service)
	}
	return nil
}

func serviceForms() list.List[schema.FormSpec] {
	return schema.FormSpecs(
		schema.FormSpec{
			Name:      "stack",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyFieldOnly,
			Fields: schema.Fields(
				schema.FieldSpec{
					Name:     "name",
					Type:     schema.TypeString,
					Required: true,
				},
			),
		},
		schema.FormSpec{
			Name:         "service",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "service",
			BodyMode:     schema.BodyFieldOnly,
			Declares:     "service",
			Fields: schema.Fields(
				schema.FieldSpec{
					Name:     "image",
					Type:     schema.TypeString,
					Required: true,
				},
				schema.FieldSpec{
					Name:     "port",
					Type:     schema.TypeInt,
					Required: true,
				},
				schema.FieldSpec{
					Name:       "depends_on",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "service"}},
					Default:    []any{},
					HasDefault: true,
				},
				schema.FieldSpec{
					Name:       "env",
					Type:       schema.MapType{Elem: schema.TypeString},
					Default:    mapping.NewOrderedMap[string, any](),
					HasDefault: true,
				},
			),
		},
	)
}

func lowerService(form compiler.HIRForm) (Service, error) {
	if form.Symbol == nil {
		return Service{}, serviceDSLErrorf("service form requires symbol label")
	}
	image, err := requiredStringField(form, "image")
	if err != nil {
		return Service{}, err
	}
	portValue, ok := form.Field("port")
	if !ok {
		return Service{}, serviceDSLErrorf("service.port is required")
	}
	port, ok := portValue.Value.(int64)
	if !ok {
		return Service{}, serviceDSLErrorf("service.port must be int")
	}
	dependsField, _ := form.Field("depends_on")
	dependsOn, err := refNames(dependsField.Value, "service")
	if err != nil {
		return Service{}, err
	}
	envField, _ := form.Field("env")
	env, err := stringMap(envField.Value)
	if err != nil {
		return Service{}, err
	}
	return Service{
		Name:      form.Symbol.Name,
		Image:     image,
		Port:      port,
		DependsOn: dependsOn,
		Env:       env,
	}, nil
}

func requiredStringField(form compiler.HIRForm, name string) (string, error) {
	field, ok := form.Field(name)
	if !ok {
		return "", serviceDSLErrorf("%s.%s is required", form.Kind, name)
	}
	value, ok := field.Value.(string)
	if !ok {
		return "", serviceDSLErrorf("%s.%s must be string", form.Kind, name)
	}
	return value, nil
}

func refNames(value any, kind string) (list.List[string], error) {
	items, ok := value.([]any)
	if !ok {
		return list.List[string]{}, serviceDSLErrorf("expected list of refs, got %T", value)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		ref, ok := item.(schema.Ref)
		if !ok {
			return list.List[string]{}, serviceDSLErrorf("expected ref<%s>, got %T", kind, item)
		}
		if ref.Kind != kind {
			return list.List[string]{}, serviceDSLErrorf("expected ref<%s>, got ref<%s>", kind, ref.Kind)
		}
		names = append(names, ref.Name)
	}
	return *list.NewList(names...), nil
}

func stringMap(value any) (mapping.OrderedMap[string, string], error) {
	out := mapping.NewOrderedMap[string, string]()
	switch items := value.(type) {
	case *mapping.OrderedMap[string, any]:
		if err := copyOrderedStringMap(out, items); err != nil {
			return mapping.OrderedMap[string, string]{}, err
		}
	case map[string]any:
		if err := copyBuiltinStringMap(out, items); err != nil {
			return mapping.OrderedMap[string, string]{}, err
		}
	default:
		return mapping.OrderedMap[string, string]{}, serviceDSLErrorf("expected string map, got %T", value)
	}
	return *out, nil
}

func copyOrderedStringMap(out *mapping.OrderedMap[string, string], items *mapping.OrderedMap[string, any]) error {
	for _, key := range items.Keys() {
		item, _ := items.Get(key)
		text, err := stringValue(item)
		if err != nil {
			return err
		}
		out.Set(key, text)
	}
	return nil
}

func copyBuiltinStringMap(out *mapping.OrderedMap[string, string], items map[string]any) error {
	keys := lo.Keys(items)
	slices.Sort(keys)
	for _, key := range keys {
		item := items[key]
		text, err := stringValue(item)
		if err != nil {
			return err
		}
		out.Set(key, text)
	}
	return nil
}

func stringValue(item any) (string, error) {
	text, ok := item.(string)
	if !ok {
		return "", serviceDSLErrorf("expected env string value, got %T", item)
	}
	return text, nil
}
