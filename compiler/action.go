// Package compiler provides plano's typed-document compiler and host extension points.
package compiler

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/plano/schema"
)

type ActionSpec struct {
	Name         string
	MinArgs      int
	MaxArgs      int
	ArgTypes     []schema.Type
	VariadicType schema.Type
	Validate     func(args []any) error
	Docs         string
}

func (c *Compiler) RegisterAction(spec ActionSpec) error {
	if spec.Name == "" {
		return errors.New("action name cannot be empty")
	}
	c.actions.Set(spec.Name, spec)
	return nil
}

func validateArity(kind, name string, minArgs, maxArgs, actual int) error {
	if actual < minArgs {
		return fmt.Errorf("%s %q requires at least %d arguments", kind, name, minArgs)
	}
	if maxArgs >= 0 && actual > maxArgs {
		return fmt.Errorf("%s %q accepts at most %d arguments", kind, name, maxArgs)
	}
	return nil
}
