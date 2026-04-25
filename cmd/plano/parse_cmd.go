package main

import (
	"go/token"
	"io"

	planofrontend "github.com/arcgolabs/plano/frontend/plano"
	"github.com/spf13/cobra"
)

func newParseCmd() *cobra.Command {
	opts := outputOptions{
		format: string(formatJSON),
		out:    "-",
	}
	cmd := &cobra.Command{
		Use:   "parse <file>",
		Short: "Parse a .plano file and print the AST",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]
			src, err := readFile(filename)
			if err != nil {
				return err
			}
			fset := token.NewFileSet()
			file, diags := planofrontend.ParseFile(fset, filename, src)
			if diags.HasError() {
				return printDiagnostics(cmd.ErrOrStderr(), fset, diags)
			}
			return withOutput(cmd.OutOrStdout(), opts.out, func(w io.Writer) error {
				return writeValue(w, file, outputFormat(opts.format))
			})
		},
	}
	bindOutputFlags(cmd, &opts, false)
	return cmd
}
