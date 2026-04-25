package main

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "plano",
		Short:         "Inspect and compile plano files",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		newExamplesCmd(),
		newParseCmd(),
		newBindCmd(),
		newCheckCmd(),
		newHIRCmd(),
		newCompileCmd(),
		newLowerCmd(),
		newValidateCmd(),
		newDiagCmd(),
	)
	return cmd
}
