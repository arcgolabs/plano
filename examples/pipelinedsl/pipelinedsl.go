// Package pipelinedsl is an example CI pipeline DSL built on top of plano.
package pipelinedsl

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/lo"
)

type Pipeline struct {
	Name   string
	Stages *mapping.OrderedMap[string, Stage]
}

type Stage struct {
	Name     string
	Needs    *list.List[string]
	Image    string
	Commands []Command
}

type Command struct {
	Name string
	Args []string
}

func Register(c *compiler.Compiler) error {
	for _, spec := range pipelineForms() {
		if err := c.RegisterForm(spec); err != nil {
			return fmt.Errorf("register form %q: %w", spec.Name, err)
		}
	}
	for _, action := range pipelineActions() {
		if err := c.RegisterAction(action); err != nil {
			return fmt.Errorf("register action %q: %w", action.Name, err)
		}
	}
	return nil
}

func Lower(hir *compiler.HIR) (*Pipeline, error) {
	project := &Pipeline{
		Stages: mapping.NewOrderedMap[string, Stage](),
	}
	for _, form := range hir.Forms {
		if err := applyRootForm(project, form); err != nil {
			return nil, err
		}
	}
	if project.Name == "" {
		return nil, errors.New("pipelinedsl: pipeline form is required")
	}
	return project, nil
}

func applyRootForm(project *Pipeline, form compiler.HIRForm) error {
	switch form.Kind {
	case "pipeline":
		name, err := requiredStringField(form, "name")
		if err != nil {
			return err
		}
		if project.Name != "" {
			return errors.New("pipelinedsl: only one pipeline form is allowed")
		}
		project.Name = name
	case "stage":
		stage, err := lowerStage(form)
		if err != nil {
			return err
		}
		project.Stages.Set(stage.Name, stage)
	}
	return nil
}

func pipelineForms() []schema.FormSpec {
	return []schema.FormSpec{
		{
			Name:      "pipeline",
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
			Name:         "stage",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "stage",
			BodyMode:     schema.BodyScript,
			Declares:     "stage",
			Fields: map[string]schema.FieldSpec{
				"needs": {
					Name:       "needs",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "stage"}},
					Default:    []any{},
					HasDefault: true,
				},
				"image": {
					Name:     "image",
					Type:     schema.TypeString,
					Required: true,
				},
			},
			NestedForms: map[string]struct{}{
				"run": {},
			},
		},
		{
			Name:      "run",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyCallOnly,
		},
	}
}

func pipelineActions() []compiler.ActionSpec {
	return []compiler.ActionSpec{
		{
			Name:         "exec",
			MinArgs:      1,
			MaxArgs:      -1,
			ArgTypes:     []schema.Type{schema.TypeString},
			VariadicType: schema.TypeString,
			Validate:     validateStringArgs("exec"),
		},
		{
			Name:     "shell",
			MinArgs:  1,
			MaxArgs:  1,
			ArgTypes: []schema.Type{schema.TypeString},
			Validate: validateStringArgs("shell"),
		},
	}
}

func lowerStage(form compiler.HIRForm) (Stage, error) {
	if form.Symbol == nil {
		return Stage{}, errors.New("pipelinedsl: stage form requires symbol label")
	}
	needsField, _ := form.Field("needs")
	needs, err := refNames(needsField.Value, "stage")
	if err != nil {
		return Stage{}, err
	}
	image, err := requiredStringField(form, "image")
	if err != nil {
		return Stage{}, err
	}
	commands, err := lowerCommands(form)
	if err != nil {
		return Stage{}, err
	}
	return Stage{
		Name:     form.Symbol.Name,
		Needs:    list.NewList(needs...),
		Image:    image,
		Commands: commands,
	}, nil
}

func requiredStringField(form compiler.HIRForm, name string) (string, error) {
	field, ok := form.Field(name)
	if !ok {
		return "", fmt.Errorf("pipelinedsl: %s.%s is required", form.Kind, name)
	}
	value, ok := field.Value.(string)
	if !ok {
		return "", fmt.Errorf("pipelinedsl: %s.%s must be string", form.Kind, name)
	}
	return value, nil
}

func lowerCommands(form compiler.HIRForm) ([]Command, error) {
	commands := make([]Command, 0, len(form.Calls))
	direct, err := callsToCommands(form.Calls)
	if err != nil {
		return nil, err
	}
	commands = append(commands, direct...)
	for _, nested := range form.Forms {
		if nested.Kind != "run" {
			continue
		}
		items, err := callsToCommands(nested.Calls)
		if err != nil {
			return nil, err
		}
		commands = append(commands, items...)
	}
	return commands, nil
}

func callsToCommands(calls []compiler.HIRCall) ([]Command, error) {
	commands := make([]Command, 0, len(calls))
	for _, call := range calls {
		args, err := stringList(lo.Map(call.Args, func(arg compiler.HIRArg, _ int) any {
			return arg.Value
		}))
		if err != nil {
			return nil, err
		}
		commands = append(commands, Command{Name: call.Name, Args: args})
	}
	return commands, nil
}

func refNames(value any, kind string) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("pipelinedsl: expected list of refs, got %T", value)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		ref, ok := item.(schema.Ref)
		if !ok {
			return nil, fmt.Errorf("pipelinedsl: expected ref<%s>, got %T", kind, item)
		}
		if ref.Kind != kind {
			return nil, fmt.Errorf("pipelinedsl: expected ref<%s>, got ref<%s>", kind, ref.Kind)
		}
		names = append(names, ref.Name)
	}
	return names, nil
}

func stringList(value any) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("pipelinedsl: expected list of strings, got %T", value)
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("pipelinedsl: expected string, got %T", item)
		}
		values = append(values, text)
	}
	return values, nil
}

func validateStringArgs(name string) func(args []any) error {
	return func(args []any) error {
		for _, arg := range args {
			if _, ok := arg.(string); !ok {
				return fmt.Errorf("pipelinedsl: action %q expects string arguments, got %T", name, arg)
			}
		}
		return nil
	}
}
