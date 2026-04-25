// Package compiler provides plano's typed-document compiler and host extension points.
package compiler

import (
	"errors"
	"fmt"
)

type ActionSpec struct {
	Name     string
	MinArgs  int
	MaxArgs  int
	Validate func(args []any) error
	Docs     string
}

func (c *Compiler) RegisterAction(spec ActionSpec) error {
	if spec.Name == "" {
		return errors.New("action name cannot be empty")
	}
	c.actions[spec.Name] = spec
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
