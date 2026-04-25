// Package builddsl is an example host DSL built on top of plano.
// It is not part of the stable plano core API.
//
//nolint:cyclop,gocognit,gocyclo,funlen,revive // The example DSL keeps its registration and lowering logic in one place.
package builddsl

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/lo"
	"github.com/samber/mo"
)

type Project struct {
	Workspace mo.Option[Workspace]
	Tasks     *mapping.OrderedMap[string, Task]
}

type Workspace struct {
	Name        string
	DefaultTask string
}

type Task struct {
	Name     string
	Deps     []string
	Outputs  []string
	Commands []Command
}

type Command struct {
	Name string
	Args []string
}

func Register(c *compiler.Compiler) error {
	for _, spec := range []schema.FormSpec{
		{
			Name:      "workspace",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyFieldOnly,
			Fields: map[string]schema.FieldSpec{
				"name": {
					Name:     "name",
					Type:     schema.TypeString,
					Required: true,
				},
				"default": {
					Name:     "default",
					Type:     schema.RefType{Kind: "task"},
					Required: true,
				},
			},
		},
		{
			Name:         "task",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "task",
			BodyMode:     schema.BodyScript,
			Declares:     "task",
			Fields: map[string]schema.FieldSpec{
				"deps": {
					Name:       "deps",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "task"}},
					Default:    []any{},
					HasDefault: true,
				},
				"outputs": {
					Name:       "outputs",
					Type:       schema.ListType{Elem: schema.TypePath},
					Default:    []any{},
					HasDefault: true,
				},
			},
			NestedForms: map[string]struct{}{
				"run": {},
			},
		},
		{
			Name:         "go.test",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "task",
			BodyMode:     schema.BodyFieldOnly,
			Declares:     "task",
			Fields: map[string]schema.FieldSpec{
				"deps": {
					Name:       "deps",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "task"}},
					Default:    []any{},
					HasDefault: true,
				},
				"packages": {
					Name:     "packages",
					Type:     schema.ListType{Elem: schema.TypePath},
					Required: true,
				},
			},
		},
		{
			Name:         "go.binary",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "task",
			BodyMode:     schema.BodyFieldOnly,
			Declares:     "task",
			Fields: map[string]schema.FieldSpec{
				"deps": {
					Name:       "deps",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "task"}},
					Default:    []any{},
					HasDefault: true,
				},
				"main": {
					Name:     "main",
					Type:     schema.TypePath,
					Required: true,
				},
				"out": {
					Name:     "out",
					Type:     schema.TypePath,
					Required: true,
				},
			},
		},
		{
			Name:      "run",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyCallOnly,
		},
	} {
		if err := c.RegisterForm(spec); err != nil {
			return fmt.Errorf("register form %q: %w", spec.Name, err)
		}
	}
	for _, action := range []compiler.ActionSpec{
		{
			Name:    "exec",
			MinArgs: 1,
			MaxArgs: -1,
			Validate: func(args []any) error {
				return validateStringArgs("exec", args)
			},
		},
		{
			Name:    "shell",
			MinArgs: 1,
			MaxArgs: 1,
			Validate: func(args []any) error {
				return validateStringArgs("shell", args)
			},
		},
	} {
		if err := c.RegisterAction(action); err != nil {
			return fmt.Errorf("register action %q: %w", action.Name, err)
		}
	}
	return nil
}

func Lower(doc *compiler.Document) (*Project, error) {
	project := &Project{
		Workspace: mo.None[Workspace](),
		Tasks:     mapping.NewOrderedMap[string, Task](),
	}

	for _, form := range doc.Forms {
		switch form.Kind {
		case "workspace":
			workspace, err := lowerWorkspace(form)
			if err != nil {
				return nil, err
			}
			if project.Workspace.IsPresent() {
				return nil, errors.New("builddsl: only one workspace form is allowed")
			}
			project.Workspace = mo.Some(workspace)
		case "task":
			task, err := lowerTask(form)
			if err != nil {
				return nil, err
			}
			project.Tasks.Set(task.Name, task)
		case "go.test":
			task, err := lowerGoTestTask(form)
			if err != nil {
				return nil, err
			}
			project.Tasks.Set(task.Name, task)
		case "go.binary":
			task, err := lowerGoBinaryTask(form)
			if err != nil {
				return nil, err
			}
			project.Tasks.Set(task.Name, task)
		}
	}

	if project.Workspace.IsAbsent() {
		return nil, errors.New("builddsl: workspace form is required")
	}
	return project, nil
}

func lowerWorkspace(form compiler.Form) (Workspace, error) {
	name, ok := form.Fields["name"].(string)
	if !ok {
		return Workspace{}, errors.New("builddsl: workspace.name must be string")
	}
	defaultTask, ok := form.Fields["default"].(schema.Ref)
	if !ok || defaultTask.Kind != "task" {
		return Workspace{}, errors.New("builddsl: workspace.default must be ref<task>")
	}
	return Workspace{
		Name:        name,
		DefaultTask: defaultTask.Name,
	}, nil
}

func lowerTask(form compiler.Form) (Task, error) {
	if form.Symbol == nil {
		return Task{}, errors.New("builddsl: task form requires symbol label")
	}

	deps, err := refNames(form.Fields["deps"], "task")
	if err != nil {
		return Task{}, err
	}
	outputs, err := stringList(form.Fields["outputs"])
	if err != nil {
		return Task{}, err
	}
	commands, err := lowerCommands(form)
	if err != nil {
		return Task{}, err
	}

	return Task{
		Name:     form.Symbol.Name,
		Deps:     deps,
		Outputs:  outputs,
		Commands: commands,
	}, nil
}

func lowerGoTestTask(form compiler.Form) (Task, error) {
	if form.Symbol == nil {
		return Task{}, errors.New("builddsl: go.test form requires symbol label")
	}
	deps, err := refNames(form.Fields["deps"], "task")
	if err != nil {
		return Task{}, err
	}
	packages, err := stringList(form.Fields["packages"])
	if err != nil {
		return Task{}, err
	}
	return Task{
		Name: form.Symbol.Name,
		Deps: deps,
		Commands: []Command{
			{
				Name: "exec",
				Args: append([]string{"go", "test"}, packages...),
			},
		},
	}, nil
}

func lowerGoBinaryTask(form compiler.Form) (Task, error) {
	if form.Symbol == nil {
		return Task{}, errors.New("builddsl: go.binary form requires symbol label")
	}
	deps, err := refNames(form.Fields["deps"], "task")
	if err != nil {
		return Task{}, err
	}
	mainPath, ok := form.Fields["main"].(string)
	if !ok {
		return Task{}, errors.New("builddsl: go.binary.main must be string")
	}
	outPath, ok := form.Fields["out"].(string)
	if !ok {
		return Task{}, errors.New("builddsl: go.binary.out must be string")
	}
	return Task{
		Name:    form.Symbol.Name,
		Deps:    deps,
		Outputs: []string{outPath},
		Commands: []Command{
			{
				Name: "exec",
				Args: []string{"go", "build", "-o", outPath, mainPath},
			},
		},
	}, nil
}

func lowerCommands(form compiler.Form) ([]Command, error) {
	commands, err := callsToCommands(form.Calls)
	if err != nil {
		return nil, err
	}
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

func callsToCommands(calls []compiler.Call) ([]Command, error) {
	out := make([]Command, 0, len(calls))
	for _, call := range calls {
		args, err := stringList(call.Args)
		if err != nil {
			return nil, err
		}
		out = append(out, Command{Name: call.Name, Args: args})
	}
	return out, nil
}

func refNames(value any, kind string) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("builddsl: expected list of refs, got %T", value)
	}
	refs := make([]schema.Ref, 0, len(items))
	for _, item := range items {
		ref, ok := item.(schema.Ref)
		if !ok {
			return nil, fmt.Errorf("builddsl: expected ref<%s>, got %T", kind, item)
		}
		if ref.Kind != kind {
			return nil, fmt.Errorf("builddsl: expected ref<%s>, got ref<%s>", kind, ref.Kind)
		}
		refs = append(refs, ref)
	}
	return lo.Map(refs, func(item schema.Ref, _ int) string {
		return item.Name
	}), nil
}

func stringList(value any) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("builddsl: expected list of strings, got %T", value)
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("builddsl: expected string, got %T", item)
		}
		values = append(values, text)
	}
	return lo.Map(values, func(item string, _ int) string {
		return item
	}), nil
}

func validateStringArgs(name string, args []any) error {
	for _, arg := range args {
		if _, ok := arg.(string); !ok {
			return fmt.Errorf("builddsl: action %q expects string arguments, got %T", name, arg)
		}
	}
	return nil
}
