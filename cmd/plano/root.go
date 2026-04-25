package main

import (
	"fmt"

	"github.com/arcgolabs/plano/compiler"
	examplebuilddsl "github.com/arcgolabs/plano/examples/builddsl"
	"github.com/spf13/cobra"
)

type compileExample string

const (
	exampleNone     compileExample = ""
	exampleBuildDSL compileExample = "builddsl"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "plano",
		Short:         "Inspect and compile plano files",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		newParseCmd(),
		newCompileCmd(),
		newLowerCmd(),
		newValidateCmd(),
		newDiagCmd(),
	)
	return cmd
}

func newCompilerForExample(example compileExample) (*compiler.Compiler, error) {
	c := compiler.New(compiler.Options{})
	switch example {
	case exampleNone:
		return c, nil
	case exampleBuildDSL:
		if err := examplebuilddsl.Register(c); err != nil {
			return nil, fmt.Errorf("register example %q: %w", example, err)
		}
		return c, nil
	default:
		return nil, fmt.Errorf("unsupported example %q", example)
	}
}
