// Package servicedsl is an example service topology DSL built on top of plano.
package servicedsl

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
)

type Stack struct {
	Name     string
	Services *mapping.OrderedMap[string, Service]
}

type Service struct {
	Name      string
	Image     string
	Port      int64
	DependsOn []string
	Env       *mapping.OrderedMap[string, string]
}

func Register(c *compiler.Compiler) error {
	for _, spec := range serviceForms() {
		if err := c.RegisterForm(spec); err != nil {
			return fmt.Errorf("register form %q: %w", spec.Name, err)
		}
	}
	return nil
}

func Lower(hir *compiler.HIR) (*Stack, error) {
	stack := &Stack{
		Services: mapping.NewOrderedMap[string, Service](),
	}
	for _, form := range hir.Forms {
		if err := applyRootForm(stack, form); err != nil {
			return nil, err
		}
	}
	if stack.Name == "" {
		return nil, errors.New("servicedsl: stack form is required")
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
			return errors.New("servicedsl: only one stack form is allowed")
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

func serviceForms() []schema.FormSpec {
	return []schema.FormSpec{
		{
			Name:      "stack",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyFieldOnly,
			Fields: map[string]schema.FieldSpec{
				"name": {
					Name:     "name",
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
		{
			Name:         "service",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "service",
			BodyMode:     schema.BodyFieldOnly,
			Declares:     "service",
			Fields: map[string]schema.FieldSpec{
				"image": {
					Name:     "image",
					Type:     schema.TypeString,
					Required: true,
				},
				"port": {
					Name:     "port",
					Type:     schema.TypeInt,
					Required: true,
				},
				"depends_on": {
					Name:       "depends_on",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "service"}},
					Default:    []any{},
					HasDefault: true,
				},
				"env": {
					Name:       "env",
					Type:       schema.MapType{Elem: schema.TypeString},
					Default:    mapping.NewOrderedMap[string, any](),
					HasDefault: true,
				},
			},
		},
	}
}

func lowerService(form compiler.HIRForm) (Service, error) {
	if form.Symbol == nil {
		return Service{}, errors.New("servicedsl: service form requires symbol label")
	}
	image, err := requiredStringField(form, "image")
	if err != nil {
		return Service{}, err
	}
	portValue, ok := form.Field("port")
	if !ok {
		return Service{}, errors.New("servicedsl: service.port is required")
	}
	port, ok := portValue.Value.(int64)
	if !ok {
		return Service{}, errors.New("servicedsl: service.port must be int")
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
		return "", fmt.Errorf("servicedsl: %s.%s is required", form.Kind, name)
	}
	value, ok := field.Value.(string)
	if !ok {
		return "", fmt.Errorf("servicedsl: %s.%s must be string", form.Kind, name)
	}
	return value, nil
}

func refNames(value any, kind string) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("servicedsl: expected list of refs, got %T", value)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		ref, ok := item.(schema.Ref)
		if !ok {
			return nil, fmt.Errorf("servicedsl: expected ref<%s>, got %T", kind, item)
		}
		if ref.Kind != kind {
			return nil, fmt.Errorf("servicedsl: expected ref<%s>, got ref<%s>", kind, ref.Kind)
		}
		names = append(names, ref.Name)
	}
	return names, nil
}

func stringMap(value any) (*mapping.OrderedMap[string, string], error) {
	out := mapping.NewOrderedMap[string, string]()
	switch items := value.(type) {
	case *mapping.OrderedMap[string, any]:
		if err := copyOrderedStringMap(out, items); err != nil {
			return nil, err
		}
	case map[string]any:
		if err := copyBuiltinStringMap(out, items); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("servicedsl: expected string map, got %T", value)
	}
	return out, nil
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
	for key, item := range items {
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
		return "", fmt.Errorf("servicedsl: expected env string value, got %T", item)
	}
	return text, nil
}
