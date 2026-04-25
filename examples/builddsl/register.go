package builddsl

import (
	"fmt"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
)

func Register(c *compiler.Compiler) error {
	for _, spec := range formSpecs() {
		if err := c.RegisterForm(spec); err != nil {
			return fmt.Errorf("register form %q: %w", spec.Name, err)
		}
	}
	for _, action := range actionSpecs() {
		if err := c.RegisterAction(action); err != nil {
			return fmt.Errorf("register action %q: %w", action.Name, err)
		}
	}
	return nil
}

func formSpecs() []schema.FormSpec {
	return []schema.FormSpec{
		workspaceFormSpec(),
		taskFormSpec(),
		goTestFormSpec(),
		goBinaryFormSpec(),
		runFormSpec(),
	}
}

func actionSpecs() []compiler.ActionSpec {
	return []compiler.ActionSpec{
		{
			Name:         "exec",
			MinArgs:      1,
			MaxArgs:      -1,
			ArgTypes:     []schema.Type{schema.TypeString},
			VariadicType: schema.TypeString,
			Validate: func(args []any) error {
				return validateStringArgs("exec", args)
			},
		},
		{
			Name:     "shell",
			MinArgs:  1,
			MaxArgs:  1,
			ArgTypes: []schema.Type{schema.TypeString},
			Validate: func(args []any) error {
				return validateStringArgs("shell", args)
			},
		},
	}
}

func workspaceFormSpec() schema.FormSpec {
	return schema.FormSpec{
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
	}
}

func taskFormSpec() schema.FormSpec {
	return schema.FormSpec{
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
	}
}

func goTestFormSpec() schema.FormSpec {
	return schema.FormSpec{
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
	}
}

func goBinaryFormSpec() schema.FormSpec {
	return schema.FormSpec{
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
	}
}

func runFormSpec() schema.FormSpec {
	return schema.FormSpec{
		Name:      "run",
		LabelKind: schema.LabelNone,
		BodyMode:  schema.BodyCallOnly,
	}
}
