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
		Use:   "examples [sample]",
		Short: "List or print embedded plano sample files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withOutput(cmd.OutOrStdout(), opts.out, func(w io.Writer) error {
				return writeExamplesOutput(w, args, outputFormat(opts.format))
			})
		},
	}
	bindOutputFlags(cmd, &opts, true)
	return cmd
}

func writeExamplesOutput(w io.Writer, args []string, format outputFormat) error {
	if len(args) == 1 {
		return writeExampleFileOutput(w, args[0], format)
	}
	if format == formatText {
		return writeTextExamples(w)
	}
	return writeValue(w, exampleViews(), format)
}

func writeExampleFileOutput(w io.Writer, name string, format outputFormat) error {
	item, err := exampleFile(name)
	if err != nil {
		return err
	}
	if format == formatText {
		return writeString(w, item.Content)
	}
	return writeValue(w, item, format)
}

func writeTextExamples(w io.Writer) error {
	views := exampleViews()
	lines := lo.Map(views.Values(), func(item exampleView, _ int) string {
		return item.Name + ": " + item.Description + " [" + item.Path + "]"
	})
	return writeString(w, strings.Join(lines, "\n")+"\n")
}
