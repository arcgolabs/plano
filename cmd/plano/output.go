package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"io"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/diag"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type outputFormat string

const (
	formatText outputFormat = "text"
	formatJSON outputFormat = "json"
	formatYAML outputFormat = "yaml"
)

type outputOptions struct {
	format string
	out    string
}

type diagnosticView struct {
	Severity  string                           `json:"severity"            yaml:"severity"`
	Code      string                           `json:"code,omitempty"      yaml:"code,omitempty"`
	Message   string                           `json:"message"             yaml:"message"`
	File      string                           `json:"file,omitempty"      yaml:"file,omitempty"`
	Line      int                              `json:"line,omitempty"      yaml:"line,omitempty"`
	Column    int                              `json:"column,omitempty"    yaml:"column,omitempty"`
	EndLine   int                              `json:"endLine,omitempty"   yaml:"endLine,omitempty"`
	EndColumn int                              `json:"endColumn,omitempty" yaml:"endColumn,omitempty"`
	Related   list.List[diagnosticRelatedView] `json:"related"             yaml:"related"`
}

type diagnosticRelatedView struct {
	Message   string `json:"message"             yaml:"message"`
	File      string `json:"file,omitempty"      yaml:"file,omitempty"`
	Line      int    `json:"line,omitempty"      yaml:"line,omitempty"`
	Column    int    `json:"column,omitempty"    yaml:"column,omitempty"`
	EndLine   int    `json:"endLine,omitempty"   yaml:"endLine,omitempty"`
	EndColumn int    `json:"endColumn,omitempty" yaml:"endColumn,omitempty"`
}

func bindOutputFlags(cmd *cobra.Command, opts *outputOptions, includeText bool) {
	cmd.Flags().StringVarP(&opts.out, "out", "o", opts.out, "write output to file instead of stdout")
	cmd.Flags().StringVar(&opts.format, "format", opts.format, formatUsage(includeText))
}

func formatUsage(includeText bool) string {
	if includeText {
		return "output format: text, json, yaml"
	}
	return "output format: json, yaml"
}

func writeValue(w io.Writer, value any, format outputFormat) error {
	switch format {
	case formatJSON:
		return writeJSON(w, value)
	case formatYAML:
		return writeYAML(w, value)
	case formatText:
		return writeText(w, value)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}

func writeJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func writeYAML(w io.Writer, value any) error {
	normalized, err := normalizeForYAML(value)
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("encode yaml: %w", err)
	}
	if err := writeBytes(w, data); err != nil {
		return err
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return writeString(w, "\n")
	}
	return nil
}

func writeText(w io.Writer, value any) error {
	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("text output requires string value, got %T", value)
	}
	return writeString(w, text+"\n")
}

func normalizeForYAML(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("normalize yaml via json marshal: %w", err)
	}
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, fmt.Errorf("normalize yaml via json unmarshal: %w", err)
	}
	return normalized, nil
}

func withOutput(defaultWriter io.Writer, path string, fn func(w io.Writer) error) (err error) {
	if path == "" || path == "-" {
		return fn(defaultWriter)
	}
	writer, err := openWriter(path)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := writer.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	return fn(writer)
}

func printDiagnostics(w io.Writer, fset *token.FileSet, diags diag.Diagnostics) error {
	for index := range diags {
		item := diags[index]
		if err := writeString(w, item.Format(fset)+"\n"); err != nil {
			return err
		}
	}
	return errors.New("compilation failed")
}

func diagnosticsToView(fset *token.FileSet, items diag.Diagnostics) *list.List[diagnosticView] {
	views := list.NewListWithCapacity[diagnosticView](len(items))
	for index := range items {
		item := items[index]
		view := diagnosticView{
			Severity: string(item.Severity),
			Code:     string(item.Code),
			Message:  item.Message,
		}
		if fset != nil && item.Pos.IsValid() {
			start := fset.Position(item.Pos)
			end := fset.Position(item.End)
			view.File = start.Filename
			view.Line = start.Line
			view.Column = start.Column
			view.EndLine = end.Line
			view.EndColumn = end.Column
		}
		view.Related = relatedDiagnosticsToView(fset, item.Related)
		views.Add(view)
	}
	return views
}

func relatedDiagnosticsToView(fset *token.FileSet, items list.List[diag.RelatedInformation]) list.List[diagnosticRelatedView] {
	views := list.NewListWithCapacity[diagnosticRelatedView](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		view := diagnosticRelatedView{Message: item.Message}
		if fset != nil && item.Pos.IsValid() {
			start := fset.Position(item.Pos)
			end := fset.Position(item.End)
			view.File = start.Filename
			view.Line = start.Line
			view.Column = start.Column
			view.EndLine = end.Line
			view.EndColumn = end.Column
		}
		views.Add(view)
	}
	return *views
}
