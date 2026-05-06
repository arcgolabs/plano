package main

import (
	"context"
	"go/token"
	"io"
	"strings"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/diag"
	"github.com/spf13/cobra"
)

type compileOptions struct {
	strict bool
	output outputOptions
}

type phaseResult struct {
	value       any
	fileSet     *token.FileSet
	diagnostics diag.Diagnostics
}

type phaseLoader func(filename string) phaseResult

func newCompileCmd() *cobra.Command {
	return newPhaseOutputCmd(
		"compile <file>",
		"Compile a .plano file and print the typed document",
		func(filename string) phaseResult {
			result := compileDetailed(filename)
			return phaseResult{
				value:       result.Document,
				fileSet:     result.FileSet,
				diagnostics: result.Diagnostics,
			}
		},
	)
}

func newBindCmd() *cobra.Command {
	return newPhaseOutputCmd(
		"bind <file>",
		"Bind declarations in a .plano file",
		func(filename string) phaseResult {
			result := bindDetailed(filename)
			return phaseResult{
				value:       result.Binding,
				fileSet:     result.FileSet,
				diagnostics: result.Diagnostics,
			}
		},
	)
}

func newCheckCmd() *cobra.Command {
	return newPhaseOutputCmd(
		"check <file>",
		"Typecheck a .plano file",
		func(filename string) phaseResult {
			result := checkDetailed(filename)
			return phaseResult{
				value: struct {
					Binding *compiler.Binding   `json:"binding" yaml:"binding"`
					Checks  *compiler.CheckInfo `json:"checks"  yaml:"checks"`
				}{
					Binding: result.Binding,
					Checks:  result.Checks,
				},
				fileSet:     result.FileSet,
				diagnostics: result.Diagnostics,
			}
		},
	)
}

func newHIRCmd() *cobra.Command {
	return newPhaseOutputCmd(
		"hir <file>",
		"Compile a .plano file and print the typed HIR",
		func(filename string) phaseResult {
			result := compileDetailed(filename)
			return phaseResult{
				value:       result.HIR,
				fileSet:     result.FileSet,
				diagnostics: result.Diagnostics,
			}
		},
	)
}

func newPhaseOutputCmd(use, short string, load phaseLoader) *cobra.Command {
	opts := compileOptions{
		output: outputOptions{
			format: string(formatJSON),
			out:    "-",
		},
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result := load(args[0])
			if shouldFail(result.diagnostics, opts.strict) {
				return printDiagnostics(cmd.ErrOrStderr(), result.fileSet, result.diagnostics)
			}
			return withOutput(cmd.OutOrStdout(), opts.output.out, func(w io.Writer) error {
				return writeValue(w, result.value, outputFormat(opts.output.format))
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
			result := compileDetailed(args[0])
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
			result := compileDetailed(args[0])
			return withOutput(cmd.OutOrStdout(), opts.output.out, func(w io.Writer) error {
				return writeDiagnosticsValue(w, result.FileSet, result.Diagnostics, outputFormat(opts.output.format))
			})
		},
	}
	bindCompilerFlags(cmd, &opts, true)
	return cmd
}

func compileDetailed(filename string) compiler.Result {
	c := compiler.New(compiler.Options{})
	return c.CompileFileDetailed(context.Background(), filename)
}

func bindDetailed(filename string) compiler.BindResult {
	c := compiler.New(compiler.Options{})
	return c.BindFileDetailed(context.Background(), filename)
}

func checkDetailed(filename string) compiler.CheckResult {
	c := compiler.New(compiler.Options{})
	return c.CheckFileDetailed(context.Background(), filename)
}

func bindCompilerFlags(cmd *cobra.Command, opts *compileOptions, includeText bool) {
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
	for index := range items {
		item := items[index]
		lines = append(lines, item.Format(fset))
	}
	return writeString(w, strings.Join(lines, "\n")+"\n")
}
