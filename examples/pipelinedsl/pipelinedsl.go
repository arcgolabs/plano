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
	Stages mapping.OrderedMap[string, Stage]
}

type Stage struct {
	Name     string
	Needs    list.List[string]
	Image    string
	Commands list.List[Command]
}

type Command struct {
	Name string
	Args list.List[string]
}

func Register(c *compiler.Compiler) error {
	if err := c.RegisterForms(pipelineForms()); err != nil {
		return fmt.Errorf("register pipelinedsl forms: %w", err)
	}
	if err := c.RegisterActions(pipelineActions()); err != nil {
		return fmt.Errorf("register pipelinedsl actions: %w", err)
	}
	return nil
}

func Lower(hir *compiler.HIR) (*Pipeline, error) {
	project := &Pipeline{}
	for idx := range hir.Forms.Len() {
		form, _ := hir.Forms.Get(idx)
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

func pipelineForms() list.List[schema.FormSpec] {
	return schema.FormSpecs(
		schema.FormSpec{
			Name:      "pipeline",
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
			Name:         "stage",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "stage",
			BodyMode:     schema.BodyScript,
			Declares:     "stage",
			Fields: schema.Fields(
				schema.FieldSpec{
					Name:       "needs",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "stage"}},
					Default:    []any{},
					HasDefault: true,
				},
				schema.FieldSpec{
					Name:     "image",
					Type:     schema.TypeString,
					Required: true,
				},
			),
			NestedForms: schema.NestedForms("run"),
		},
		schema.FormSpec{
			Name:      "run",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyCallOnly,
		},
	)
}

func pipelineActions() list.List[compiler.ActionSpec] {
	return compiler.ActionSpecs(
		compiler.ActionSpec{
			Name:         "exec",
			MinArgs:      1,
			MaxArgs:      -1,
			ArgTypes:     schema.Types(schema.TypeString),
			VariadicType: schema.TypeString,
			Validate:     validateStringArgs("exec"),
		},
		compiler.ActionSpec{
			Name:     "shell",
			MinArgs:  1,
			MaxArgs:  1,
			ArgTypes: schema.Types(schema.TypeString),
			Validate: validateStringArgs("shell"),
		},
	)
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
		Needs:    needs,
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

func lowerCommands(form compiler.HIRForm) (list.List[Command], error) {
	commands := list.NewListWithCapacity[Command](form.Calls.Len())
	direct, err := callsToCommands(form.Calls)
	if err != nil {
		return list.List[Command]{}, err
	}
	commands.Merge(&direct)
	for idx := range form.Forms.Len() {
		nested, _ := form.Forms.Get(idx)
		if nested.Kind != "run" {
			continue
		}
		items, err := callsToCommands(nested.Calls)
		if err != nil {
			return list.List[Command]{}, err
		}
		commands.Merge(&items)
	}
	return *commands, nil
}

func callsToCommands(calls list.List[compiler.HIRCall]) (list.List[Command], error) {
	commands := list.NewListWithCapacity[Command](calls.Len())
	for idx := range calls.Len() {
		call, _ := calls.Get(idx)
		args, err := stringList(lo.Map(call.Args.Values(), func(arg compiler.HIRArg, _ int) any {
			return arg.Value
		}))
		if err != nil {
			return list.List[Command]{}, err
		}
		commands.Add(Command{Name: call.Name, Args: args})
	}
	return *commands, nil
}

func refNames(value any, kind string) (list.List[string], error) {
	items, ok := value.([]any)
	if !ok {
		return list.List[string]{}, fmt.Errorf("pipelinedsl: expected list of refs, got %T", value)
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		ref, ok := item.(schema.Ref)
		if !ok {
			return list.List[string]{}, fmt.Errorf("pipelinedsl: expected ref<%s>, got %T", kind, item)
		}
		if ref.Kind != kind {
			return list.List[string]{}, fmt.Errorf("pipelinedsl: expected ref<%s>, got ref<%s>", kind, ref.Kind)
		}
		names = append(names, ref.Name)
	}
	return *list.NewList(names...), nil
}

func stringList(value any) (list.List[string], error) {
	items, ok := value.([]any)
	if !ok {
		return list.List[string]{}, fmt.Errorf("pipelinedsl: expected list of strings, got %T", value)
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return list.List[string]{}, fmt.Errorf("pipelinedsl: expected string, got %T", item)
		}
		values = append(values, text)
	}
	return *list.NewList(values...), nil
}

func validateStringArgs(name string) func(args list.List[any]) error {
	return func(args list.List[any]) error {
		for _, arg := range args.Values() {
			if _, ok := arg.(string); !ok {
				return fmt.Errorf("pipelinedsl: action %q expects string arguments, got %T", name, arg)
			}
		}
		return nil
	}
}
