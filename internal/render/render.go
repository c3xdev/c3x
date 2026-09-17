// Package render formats Estimates and Diffs for human and machine
// consumption. It owns no business logic — every renderer is a pure
// function from domain types + Format to a string.
//
// Three formats are supported:
//
//	Text      — terminal-friendly with box-drawing characters
//	Markdown  — flat, always-expanded tables (RenderMarkdown). The
//	            collapsible summary+<details> comment layout that
//	            `c3x comment` posts is RenderMarkdownComment.
//	JSON      — machine-readable, structurally identical to
//	            domain.Estimate
//
// Adding a format is one entry in [ParseFormat] + one Render method.
// Existing callers don't change because they go through [Render].
package render

import (
	"fmt"
	"strings"
)

// Format is the renderer choice. The zero value is FormatText so a
// missing flag does the right thing.
type Format int

const (
	FormatText Format = iota
	FormatMarkdown
	FormatJSON
	// FormatJUnit emits a JUnit-XML report — one <testcase> per
	// resource, <failure> when a budget threshold is breached. Used
	// to surface c3x results in CI dashboards (GitLab, CircleCI,
	// Jenkins) that already consume JUnit.
	FormatJUnit
	// FormatHTML produces a self-contained styled HTML report —
	// inline CSS, no JS, no external assets. Suitable for emailing,
	// attaching to a ticket, or hosting as a static artifact.
	FormatHTML
	// FormatCSV emits a per-LineItem CSV with a trailing TOTAL
	// row. The shape suits spreadsheet workflows where users
	// pivot/sum by kind, dimension, or source.
	FormatCSV
	// FormatSARIF emits a SARIF v2.1.0 document with one result
	// per resource, severity bucketed by cost magnitude. GitHub
	// code-scanning, GitLab security dashboards, and similar
	// surfaces consume SARIF natively.
	FormatSARIF
)

// String returns the canonical CLI name for the format. Symmetric
// with [ParseFormat] for round-trippability.
func (f Format) String() string {
	switch f {
	case FormatText:
		return "text"
	case FormatMarkdown:
		return "markdown"
	case FormatJSON:
		return "json"
	case FormatJUnit:
		return "junit"
	case FormatHTML:
		return "html"
	case FormatCSV:
		return "csv"
	case FormatSARIF:
		return "sarif"
	default:
		return "unknown"
	}
}

// ParseFormat converts a CLI string into a Format. Unknown values
// return an error so the caller can surface a friendly diagnostic
// rather than silently defaulting.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "text":
		return FormatText, nil
	case "markdown", "md":
		return FormatMarkdown, nil
	case "json":
		return FormatJSON, nil
	case "junit", "junit-xml":
		return FormatJUnit, nil
	case "html":
		return FormatHTML, nil
	case "csv":
		return FormatCSV, nil
	case "sarif":
		return FormatSARIF, nil
	default:
		return FormatText, fmt.Errorf("unknown format %q (want text|markdown|json|junit|html|csv|sarif)", s)
	}
}
