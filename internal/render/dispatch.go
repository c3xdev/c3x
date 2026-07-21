package render

import (
	"fmt"

	"github.com/c3xdev/c3x/internal/domain"
)

// Render dispatches an Estimate to the right format-specific renderer.
// Callers that don't want to switch on Format themselves use this.
func Render(est domain.Estimate, f Format) (string, error) {
	switch f {
	case FormatText:
		return RenderText(est), nil
	case FormatMarkdown:
		return RenderMarkdown(est), nil
	case FormatJSON:
		return RenderJSON(est)
	case FormatJUnit:
		return RenderJUnit(est)
	case FormatHTML:
		return RenderHTML(est)
	case FormatCSV:
		return RenderCSV(est)
	case FormatSARIF:
		return RenderSARIF(est)
	default:
		return "", fmt.Errorf("unsupported format %v", f)
	}
}

// RenderDelta dispatches an Estimate to a delta-aware renderer that
// shows only changed resources (from plan JSON) with +/~/- markers.
// Falls back to the full estimate renderer for formats that don't
// have a delta-specific view.
func RenderDelta(est domain.Estimate, f Format) (string, error) {
	switch f {
	case FormatText:
		return RenderTextDelta(est), nil
	default:
		// Other formats don't have delta-specific rendering yet;
		// fall back to the standard renderer.
		return Render(est, f)
	}
}

// RenderDiff dispatches a Diff to the right format-specific renderer.
// Every format the estimate renderer supports works here too.
func RenderDiff(d domain.Diff, f Format) (string, error) {
	switch f {
	case FormatText:
		return RenderTextDiff(d), nil
	case FormatMarkdown:
		return RenderMarkdownDiff(d), nil
	case FormatJSON:
		return RenderJSONDiff(d)
	case FormatJUnit:
		return RenderJUnitDiff(d)
	case FormatHTML:
		return RenderHTMLDiff(d)
	case FormatCSV:
		return RenderCSVDiff(d)
	case FormatSARIF:
		return RenderSARIFDiff(d)
	default:
		return "", fmt.Errorf("unsupported format %v", f)
	}
}
