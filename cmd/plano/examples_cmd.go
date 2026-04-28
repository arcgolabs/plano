package main

import (
	"io"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func newExamplesCmd() *cobra.Command {
	opts := outputOptions{
		format: string(formatText),
		out:    "-",
	}
	cmd := &cobra.Command{
		Use:   "examples",
		Short: "List bundled example host DSLs and sample files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withOutput(cmd.OutOrStdout(), opts.out, func(w io.Writer) error {
				if outputFormat(opts.format) == formatText {
					return writeTextExamples(w)
				}
				return writeValue(w, exampleViews(), outputFormat(opts.format))
			})
		},
	}
	bindOutputFlags(cmd, &opts, true)
	return cmd
}

func writeTextExamples(w io.Writer) error {
	views := exampleViews()
	lines := lo.Map(views.Values(), func(item exampleView, _ int) string {
		return item.Name + ": " + item.Description + " [" + strings.Join(item.Samples.Values(), ", ") + "]"
	})
	return writeString(w, strings.Join(lines, "\n")+"\n")
}
