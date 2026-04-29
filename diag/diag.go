// Package diag defines diagnostics emitted by the plano frontend and compiler.
package diag

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/arcgolabs/collectionx/list"
)

type Severity string
type Code string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

const (
	CodeReadFailure         Code = "read-failure"
	CodeImportCycle         Code = "import-cycle"
	CodeDuplicateDefinition Code = "duplicate-definition"
	CodeUnknownForm         Code = "unknown-form"
	CodeUnknownFunction     Code = "unknown-function"
	CodeUnknownAction       Code = "unknown-action"
	CodeUndefinedName       Code = "undefined-name"
	CodeTypeMismatch        Code = "type-mismatch"
)

type RelatedInformation struct {
	Message string
	Pos     token.Pos
	End     token.Pos
}

type Diagnostic struct {
	Severity Severity
	Code     Code
	Message  string
	Pos      token.Pos
	End      token.Pos
	Related  list.List[RelatedInformation]
}

func (d Diagnostic) Format(fset *token.FileSet) string {
	head := d.formatHead(fset)
	if d.Related.Len() == 0 {
		return head
	}
	lines := []string{head}
	for index := range d.Related.Len() {
		item, _ := d.Related.Get(index)
		lines = append(lines, "  note: "+formatRelated(fset, item))
	}
	return strings.Join(lines, "\n")
}

type Diagnostics []Diagnostic

func (d *Diagnostics) Add(severity Severity, pos, end token.Pos, message string) {
	d.AddCode(severity, "", pos, end, message)
}

func (d *Diagnostics) AddCode(severity Severity, code Code, pos, end token.Pos, message string) {
	*d = append(*d, Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  message,
		Pos:      pos,
		End:      end,
	})
}

func (d *Diagnostics) AddError(pos, end token.Pos, message string) {
	d.Add(SeverityError, pos, end, message)
}

func (d *Diagnostics) AddErrorCode(code Code, pos, end token.Pos, message string) {
	d.AddCode(SeverityError, code, pos, end, message)
}

func (d *Diagnostics) AddRelated(
	severity Severity,
	code Code,
	pos token.Pos,
	end token.Pos,
	message string,
	related ...RelatedInformation,
) {
	item := Diagnostic{
		Severity: severity,
		Code:     code,
		Message:  message,
		Pos:      pos,
		End:      end,
	}
	if len(related) > 0 {
		item.Related = *list.NewList(related...)
	}
	*d = append(*d, item)
}

func (d *Diagnostics) AddErrorRelated(code Code, pos, end token.Pos, message string, related ...RelatedInformation) {
	d.AddRelated(SeverityError, code, pos, end, message, related...)
}

func (d *Diagnostics) Append(other Diagnostics) {
	*d = append(*d, other...)
}

func (d Diagnostics) HasError() bool {
	for index := range d {
		item := d[index]
		if item.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (d Diagnostics) Error() string {
	if len(d) == 0 {
		return ""
	}
	var parts []string
	for index := range d {
		item := d[index]
		parts = append(parts, item.Message)
	}
	return strings.Join(parts, "; ")
}

func (d Diagnostic) formatHead(fset *token.FileSet) string {
	suffix := ""
	if d.Code != "" {
		suffix = " [" + string(d.Code) + "]"
	}
	if fset == nil || !d.Pos.IsValid() {
		return fmt.Sprintf("%s%s: %s", d.Severity, suffix, d.Message)
	}
	pos := fset.Position(d.Pos)
	if pos.IsValid() {
		return fmt.Sprintf("%s:%d:%d: %s%s: %s", pos.Filename, pos.Line, pos.Column, d.Severity, suffix, d.Message)
	}
	return fmt.Sprintf("%s%s: %s", d.Severity, suffix, d.Message)
}

func formatRelated(fset *token.FileSet, item RelatedInformation) string {
	if fset == nil || !item.Pos.IsValid() {
		return item.Message
	}
	pos := fset.Position(item.Pos)
	if pos.IsValid() {
		return fmt.Sprintf("%s:%d:%d: %s", pos.Filename, pos.Line, pos.Column, item.Message)
	}
	return item.Message
}
