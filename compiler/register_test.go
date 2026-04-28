package compiler_test

import (
	"strings"
	"testing"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
)

func TestRegisterFormsAndActions(t *testing.T) {
	c := compiler.New(compiler.Options{})
	err := c.RegisterForms(schema.FormSpecs(
		schema.FormSpec{
			Name:      "workspace",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyFieldOnly,
		},
		schema.FormSpec{
			Name:      "run",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyCallOnly,
		},
	))
	if err != nil {
		t.Fatal(err)
	}
	err = c.RegisterActions(compiler.ActionSpecs(
		compiler.ActionSpec{
			Name:     "exec",
			MinArgs:  1,
			MaxArgs:  -1,
			ArgTypes: schema.Types(schema.TypeString),
			Validate: func(args list.List[any]) error {
				return nil
			},
		},
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.FormSpec("workspace"); !ok {
		t.Fatal("expected workspace form spec")
	}
	if _, ok := c.FormSpec("run"); !ok {
		t.Fatal("expected run form spec")
	}
	if _, ok := c.ActionSpec("exec"); !ok {
		t.Fatal("expected exec action spec")
	}
}

func TestRegisterFormsAndActionsWrapErrors(t *testing.T) {
	c := compiler.New(compiler.Options{})
	if err := c.RegisterForms(schema.FormSpecs(schema.FormSpec{})); err == nil || !strings.Contains(err.Error(), `register form ""`) {
		t.Fatalf("register forms error = %v", err)
	}
	if err := c.RegisterActions(compiler.ActionSpecs(compiler.ActionSpec{})); err == nil || !strings.Contains(err.Error(), `register action ""`) {
		t.Fatalf("register actions error = %v", err)
	}
}
