package main

import "github.com/spf13/cobra"

func buildRootCmd(runner *compilerRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "plano",
		Short:         "Inspect and compile plano files",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		newExamplesCmd(),
		newVersionCmd(),
		newParseCmd(),
		newBindCmd(runner),
		newCheckCmd(runner),
		newHIRCmd(runner),
		newCompileCmd(runner),
		newValidateCmd(runner),
		newDiagCmd(runner),
	)
	return cmd
}
