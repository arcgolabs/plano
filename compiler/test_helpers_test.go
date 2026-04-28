package compiler_test

import (
	"errors"
	"testing"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
)

func newTestCompiler(t *testing.T) *compiler.Compiler {
	t.Helper()

	c := compiler.New(compiler.Options{
		LookupEnv: func(string) (string, bool) { return "", false },
	})
	registerForm(t, c, schema.FormSpec{
		Name:      "workspace",
		LabelKind: schema.LabelNone,
		BodyMode:  schema.BodyFieldOnly,
		Fields: schema.Fields(
			schema.FieldSpec{
				Name:     "name",
				Type:     schema.TypeString,
				Required: true,
			},
			schema.FieldSpec{
				Name:     "default",
				Type:     schema.RefType{Kind: "task"},
				Required: true,
			},
		),
	})
	registerForm(t, c, schema.FormSpec{
		Name:         "task",
		LabelKind:    schema.LabelSymbol,
		BodyMode:     schema.BodyScript,
		LabelRefKind: "task",
		Declares:     "task",
		Fields: schema.Fields(
			schema.FieldSpec{
				Name:       "deps",
				Type:       schema.ListType{Elem: schema.RefType{Kind: "task"}},
				Default:    []any{},
				HasDefault: true,
			},
			schema.FieldSpec{
				Name:       "outputs",
				Type:       schema.ListType{Elem: schema.TypePath},
				Default:    []any{},
				HasDefault: true,
			},
		),
		NestedForms: schema.NestedForms("run"),
	})
	registerForm(t, c, schema.FormSpec{
		Name:      "run",
		LabelKind: schema.LabelNone,
		BodyMode:  schema.BodyCallOnly,
	})
	registerAction(t, c, compiler.ActionSpec{
		Name:         "exec",
		MinArgs:      1,
		MaxArgs:      -1,
		ArgTypes:     []schema.Type{schema.TypeString},
		VariadicType: schema.TypeString,
		Validate: func(args []any) error {
			for _, arg := range args {
				if _, ok := arg.(string); !ok {
					return errors.New("exec expects string arguments")
				}
			}
			return nil
		},
	})
	return c
}

func registerForm(t *testing.T, c *compiler.Compiler, spec schema.FormSpec) {
	t.Helper()
	if err := c.RegisterForm(spec); err != nil {
		t.Fatal(err)
	}
}

func registerAction(t *testing.T, c *compiler.Compiler, spec compiler.ActionSpec) {
	t.Helper()
	if err := c.RegisterAction(spec); err != nil {
		t.Fatal(err)
	}
}
