package main

import (
	"context"
	"errors"
	"fmt"
	"go/token"
	"io"
	"strings"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/diag"
	examplebuilddsl "github.com/arcgolabs/plano/examples/builddsl"
	"github.com/spf13/cobra"
)

type compileOptions struct {
	example string
	strict  bool
	output  outputOptions
}

func newCompileCmd() *cobra.Command {
	opts := compileOptions{
		output: outputOptions{
			format: string(formatJSON),
			out:    "-",
		},
	}
	cmd := &cobra.Command{
		Use:   "compile <file>",
		Short: "Compile a .plano file and print the typed document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := compileDetailed(args[0], opts.example)
			if err != nil {
				return err
			}
			if shouldFail(result.Diagnostics, opts.strict) {
				return printDiagnostics(cmd.ErrOrStderr(), result.FileSet, result.Diagnostics)
			}
			return withOutput(cmd.OutOrStdout(), opts.output.out, func(w io.Writer) error {
				return writeValue(w, result.Document, outputFormat(opts.output.format))
			})
		},
	}
	bindCompilerFlags(cmd, &opts, false)
	return cmd
}

func newLowerCmd() *cobra.Command {
	opts := compileOptions{
		output: outputOptions{
			format: string(formatJSON),
			out:    "-",
		},
	}
	cmd := &cobra.Command{
		Use:   "lower <file>",
		Short: "Compile and lower a .plano file using an example host DSL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if compileExample(opts.example) == exampleNone {
				return errors.New("lower requires --example")
			}
			result, err := compileDetailed(args[0], opts.example)
			if err != nil {
				return err
			}
			if shouldFail(result.Diagnostics, opts.strict) {
				return printDiagnostics(cmd.ErrOrStderr(), result.FileSet, result.Diagnostics)
			}
			lowered, err := lowerDocument(result.Document, compileExample(opts.example))
			if err != nil {
				return err
			}
			return withOutput(cmd.OutOrStdout(), opts.output.out, func(w io.Writer) error {
				return writeValue(w, lowered, outputFormat(opts.output.format))
			})
		},
	}
	bindCompilerFlags(cmd, &opts, false)
	return cmd
}

func newValidateCmd() *cobra.Command {
	opts := compileOptions{
		output: outputOptions{
			format: string(formatText),
			out:    "-",
		},
	}
	cmd := &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate a .plano file by compiling it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := compileDetailed(args[0], opts.example)
			if err != nil {
				return err
			}
			if shouldFail(result.Diagnostics, opts.strict) {
				return printDiagnostics(cmd.ErrOrStderr(), result.FileSet, result.Diagnostics)
			}
			return withOutput(cmd.OutOrStdout(), opts.output.out, func(w io.Writer) error {
				return writeValidateResult(w, args[0], outputFormat(opts.output.format))
			})
		},
	}
	bindCompilerFlags(cmd, &opts, true)
	return cmd
}

func newDiagCmd() *cobra.Command {
	opts := compileOptions{
		output: outputOptions{
			format: string(formatText),
			out:    "-",
		},
	}
	cmd := &cobra.Command{
		Use:   "diag <file>",
		Short: "Print diagnostics for a .plano file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := compileDetailed(args[0], opts.example)
			if err != nil {
				return err
			}
			return withOutput(cmd.OutOrStdout(), opts.output.out, func(w io.Writer) error {
				return writeDiagnosticsValue(w, result.FileSet, result.Diagnostics, outputFormat(opts.output.format))
			})
		},
	}
	bindCompilerFlags(cmd, &opts, true)
	return cmd
}

func compileDetailed(filename, example string) (compiler.Result, error) {
	c, err := newCompilerForExample(compileExample(example))
	if err != nil {
		return compiler.Result{}, err
	}
	return c.CompileFileDetailed(context.Background(), filename), nil
}

func lowerDocument(doc *compiler.Document, example compileExample) (any, error) {
	switch example {
	case exampleBuildDSL:
		project, err := examplebuilddsl.Lower(doc)
		if err != nil {
			return nil, fmt.Errorf("lower with %q example: %w", example, err)
		}
		return project, nil
	case exampleNone:
		return nil, errors.New("lower requires --example")
	default:
		return nil, fmt.Errorf("unsupported example %q", example)
	}
}

func bindCompilerFlags(cmd *cobra.Command, opts *compileOptions, includeText bool) {
	cmd.Flags().StringVar(&opts.example, "example", "", "register an example host DSL (currently: builddsl)")
	cmd.Flags().BoolVar(&opts.strict, "strict", false, "fail on any diagnostics, not only errors")
	bindOutputFlags(cmd, &opts.output, includeText)
}

func shouldFail(items diag.Diagnostics, strict bool) bool {
	if len(items) == 0 {
		return false
	}
	return strict || items.HasError()
}

func writeValidateResult(w io.Writer, filename string, format outputFormat) error {
	result := struct {
		Status string `json:"status" yaml:"status"`
		File   string `json:"file"   yaml:"file"`
	}{
		Status: "valid",
		File:   filename,
	}
	if format == formatText {
		return writeValue(w, "valid", format)
	}
	return writeValue(w, result, format)
}

func writeDiagnosticsValue(w io.Writer, fset *token.FileSet, items diag.Diagnostics, format outputFormat) error {
	if format == formatText {
		return writeDiagnosticsText(w, fset, items)
	}
	return writeValue(w, diagnosticsToView(fset, items), format)
}

func writeDiagnosticsText(w io.Writer, fset *token.FileSet, items diag.Diagnostics) error {
	if len(items) == 0 {
		return writeString(w, "no diagnostics\n")
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, item.Format(fset))
	}
	return writeString(w, strings.Join(lines, "\n")+"\n")
}
