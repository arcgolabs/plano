package compiler_test

import (
	"errors"
	"testing"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
)

func newTestCompiler(t *testing.T) *compiler.Compiler {
	t.Helper()
	return newRegisteredCompiler(t)
}

func newRegisteredCompiler(tb testing.TB) *compiler.Compiler {
	tb.Helper()
	c := compiler.New(compiler.Options{
		LookupEnv: func(string) (string, bool) { return "", false },
	})
	registerForms(tb, c, schema.FormSpecs(
		schema.FormSpec{
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
		},
		schema.FormSpec{
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
		},
		schema.FormSpec{
			Name:      "run",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyCallOnly,
		},
	))
	registerActions(tb, c, compiler.ActionSpecs(
		compiler.ActionSpec{
			Name:         "exec",
			MinArgs:      1,
			MaxArgs:      -1,
			ArgTypes:     schema.Types(schema.TypeString),
			VariadicType: schema.TypeString,
			Validate: func(args list.List[any]) error {
				for _, arg := range args.Values() {
					if _, ok := arg.(string); !ok {
						return errors.New("exec expects string arguments")
					}
				}
				return nil
			},
		},
	))
	return c
}

func registerForms(tb testing.TB, c *compiler.Compiler, specs list.List[schema.FormSpec]) {
	tb.Helper()
	if err := c.RegisterForms(specs); err != nil {
		tb.Fatal(err)
	}
}

func registerActions(tb testing.TB, c *compiler.Compiler, specs list.List[compiler.ActionSpec]) {
	tb.Helper()
	if err := c.RegisterActions(specs); err != nil {
		tb.Fatal(err)
	}
}
