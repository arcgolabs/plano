// Package diag defines diagnostics emitted by the plano frontend and compiler.
package diag

import (
	"fmt"
	"go/token"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	Severity Severity
	Message  string
	Pos      token.Pos
	End      token.Pos
}

func (d Diagnostic) Format(fset *token.FileSet) string {
	if fset == nil || !d.Pos.IsValid() {
		return fmt.Sprintf("%s: %s", d.Severity, d.Message)
	}
	pos := fset.Position(d.Pos)
	if pos.IsValid() {
		return fmt.Sprintf("%s:%d:%d: %s: %s", pos.Filename, pos.Line, pos.Column, d.Severity, d.Message)
	}
	return fmt.Sprintf("%s: %s", d.Severity, d.Message)
}

type Diagnostics []Diagnostic

func (d *Diagnostics) Add(severity Severity, pos, end token.Pos, message string) {
	*d = append(*d, Diagnostic{
		Severity: severity,
		Message:  message,
		Pos:      pos,
		End:      end,
	})
}

func (d *Diagnostics) AddError(pos, end token.Pos, message string) {
	d.Add(SeverityError, pos, end, message)
}

func (d *Diagnostics) Append(other Diagnostics) {
	*d = append(*d, other...)
}

func (d Diagnostics) HasError() bool {
	for _, item := range d {
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
	for _, item := range d {
		parts = append(parts, item.Message)
	}
	return strings.Join(parts, "; ")
}
