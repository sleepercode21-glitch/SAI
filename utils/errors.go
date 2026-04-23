package utils

import (
	"fmt"

	"github.com/sleepercode/sai/ast"
)

// Diagnostic is a structured compiler error.
type Diagnostic struct {
	Stage   string   `json:"stage"`
	Message string   `json:"message"`
	Span    ast.Span `json:"span"`
}

func (d Diagnostic) Error() string {
	return fmt.Sprintf("%s error at %d:%d: %s", d.Stage, d.Span.Start.Line, d.Span.Start.Column, d.Message)
}

// Diagnostics aggregates multiple compiler errors without losing location data.
type Diagnostics struct {
	Items []Diagnostic `json:"items"`
}

func (d *Diagnostics) Add(stage, message string, span ast.Span) {
	d.Items = append(d.Items, Diagnostic{
		Stage:   stage,
		Message: message,
		Span:    span,
	})
}

func (d *Diagnostics) AddDiagnostic(diag Diagnostic) {
	d.Items = append(d.Items, diag)
}

func (d *Diagnostics) HasErrors() bool {
	return len(d.Items) > 0
}

func (d *Diagnostics) Error() string {
	if len(d.Items) == 0 {
		return "no diagnostics"
	}
	return d.Items[0].Error()
}
