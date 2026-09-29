package render

// Diff renderers for the machine / report formats: JUnit, HTML, CSV,
// SARIF. The estimate renderers came first; these complete the
// matrix so `c3x diff --format <any>` works everywhere
// `c3x estimate --format <any>` does.
//
// Per-format semantics for a DIFF (as opposed to an estimate):
//   - JUnit:  one <testcase> per changed resource; the delta rides
//             in `time`; added/removed resources annotate the name.
//   - CSV:    one row per delta with baseline/current/delta columns.
//   - SARIF:  severity follows the delta sign and magnitude — cost
//             increases are findings, decreases are notes.
//   - HTML:   compact change table with the same shading scale the
//             estimate report uses, applied to |delta|.

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
)

// RenderJUnitDiff emits one <testcase> per resource delta. There are
// no failures by default — the budget gate lives in `--budget-delta`,
// not in the report — but downstream dashboards get the per-resource
// delta as the case duration for trend graphs.
func RenderJUnitDiff(d domain.Diff) (string, error) {
	suites := junitTestsuites{
		Name:  "c3x",
		Tests: len(d.Resources),
		Time:  "0",
		Suites: []junitTestsuite{
			{
				Name:  "c3x.diff",
				Tests: len(d.Resources),
				Time:  "0",
				Properties: &junitProperties{Props: []junitProperty{
					{Name: "c3x.baseline_total", Value: d.BaselineTotal.Round(2).String()},
					{Name: "c3x.current_total", Value: d.CurrentTotal.Round(2).String()},
					{Name: "c3x.total_delta", Value: d.TotalDelta.Round(2).String()},
					{Name: "c3x.currency", Value: d.Currency.String()},
				}},
				Cases: make([]junitTestcase, 0, len(d.Resources)),
			},
		},
	}
	for _, r := range d.Resources {
		suites.Suites[0].Cases = append(suites.Suites[0].Cases, junitTestcase{
			Classname: r.Resource.Kind,
			Name:      fmt.Sprintf("%s [%s]", r.Resource.Name, deltaKindLabel(r.Kind)),
			Time:      r.Delta.Round(2).String(),
		})
	}
	out, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JUnit diff: %w", err)
	}
	return xml.Header + string(out) + "\n", nil
}

// RenderCSVDiff emits one row per delta plus a trailing TOTAL row.
func RenderCSVDiff(d domain.Diff) (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{
		"resource", "kind", "change", "baseline", "current", "delta", "currency",
	}); err != nil {
		return "", err
	}
	currency := d.Currency.String()
	for _, r := range d.Resources {
		_ = w.Write([]string{
			r.Resource.Label(),
			r.Resource.Kind,
			deltaKindLabel(r.Kind),
			r.Baseline.Round(2).String(),
			r.Current.Round(2).String(),
			r.Delta.Round(2).String(),
			currency,
		})
	}
	_ = w.Write([]string{
		"PROJECT", "", "TOTAL",
		d.BaselineTotal.Round(2).String(),
		d.CurrentTotal.Round(2).String(),
		d.TotalDelta.Round(2).String(),
		currency,
	})
	w.Flush()
	return buf.String(), w.Error()
}

// RenderSARIFDiff maps deltas to findings: cost increases are
// warnings (error above $100/mo added), decreases and no-ops are
// notes. The result set is consumable by the same code-scanning
// surfaces as the estimate SARIF.
func RenderSARIFDiff(d domain.Diff) (string, error) {
	doc := sarifDoc{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{
			{
				Tool: sarifTool{Driver: sarifDriver{
					Name:           "c3x",
					InformationURI: "https://c3x.dev",
					Rules: []sarifRule{
						{
							ID: "diff.note", Name: "CostUnchangedOrReduced",
							ShortDescription: sarifMessage{Text: "Cost unchanged or reduced"},
						},
						{
							ID: "diff.warning", Name: "CostIncreased",
							ShortDescription: sarifMessage{Text: "Monthly cost increased"},
						},
						{
							ID: "diff.error", Name: "LargeCostIncrease",
							ShortDescription: sarifMessage{Text: "Monthly cost increased by more than $100"},
						},
					},
				}},
				Results: make([]sarifResult, 0, len(d.Resources)),
			},
		},
	}
	sym := d.Currency.Symbol()
	for _, r := range d.Resources {
		delta, _ := r.Delta.Float64()
		level, rule := "note", "diff.note"
		switch {
		case delta >= 100:
			level, rule = "error", "diff.error"
		case delta > 0:
			level, rule = "warning", "diff.warning"
		}
		doc.Runs[0].Results = append(doc.Runs[0].Results, sarifResult{
			RuleID: rule,
			Level:  level,
			Message: sarifMessage{Text: fmt.Sprintf("%s.%s (%s): %s%s → %s%s (Δ %s%s/mo)",
				r.Resource.Kind, r.Resource.Name, deltaKindLabel(r.Kind),
				sym, r.Baseline.Round(2), sym, r.Current.Round(2), sym, r.Delta.Round(2))},
			Properties: map[string]any{
				"kind":     r.Resource.Kind,
				"name":     r.Resource.Name,
				"change":   deltaKindLabel(r.Kind),
				"baseline": r.Baseline.Round(2).String(),
				"current":  r.Current.Round(2).String(),
				"delta":    r.Delta.Round(2).String(),
				"currency": d.Currency.String(),
			},
		})
	}
	buf, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(buf) + "\n", nil
}

// RenderHTMLDiff produces a self-contained HTML change report using
// the same inline-CSS conventions as the estimate report.
func RenderHTMLDiff(d domain.Diff) (string, error) {
	maxAbs := decimal.Zero
	for _, r := range d.Resources {
		if a := r.Delta.Abs(); a.GreaterThan(maxAbs) {
			maxAbs = a
		}
	}
	data := htmlDiffData{
		Currency:      d.Currency.Symbol(),
		BaselineTotal: d.BaselineTotal.Round(2).String(),
		CurrentTotal:  d.CurrentTotal.Round(2).String(),
		TotalDelta:    signed(d.Currency.Symbol(), d.TotalDelta.Round(2).String()),
		Rows:          make([]htmlDiffRow, 0, len(d.Resources)),
	}
	for _, r := range d.Resources {
		data.Rows = append(data.Rows, htmlDiffRow{
			Label:    r.Resource.Label(),
			Change:   deltaKindLabel(r.Kind),
			Baseline: r.Baseline.Round(2).String(),
			Current:  r.Current.Round(2).String(),
			Delta:    signed(d.Currency.Symbol(), r.Delta.Round(2).String()),
			Shade:    shadeFor(r.Delta.Abs(), maxAbs),
		})
	}
	var buf bytes.Buffer
	if err := htmlDiffTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute HTML diff template: %w", err)
	}
	return buf.String(), nil
}

func deltaKindLabel(k domain.DeltaKind) string {
	switch k {
	case domain.DeltaAdded:
		return "added"
	case domain.DeltaRemoved:
		return "removed"
	case domain.DeltaModified:
		return "modified"
	default:
		return "unchanged"
	}
}

type htmlDiffData struct {
	Currency      string
	BaselineTotal string
	CurrentTotal  string
	TotalDelta    string
	Rows          []htmlDiffRow
}

type htmlDiffRow struct {
	Label    string
	Change   string
	Baseline string
	Current  string
	Delta    string
	Shade    string
}

var htmlDiffTmpl = template.Must(template.New("c3x-diff").Parse(`<!doctype html>
<html lang="en"><head>
<meta charset="utf-8"><title>c3x cost diff</title>
<style>
  :root { color-scheme: light dark; --fg:#1c1c1c; --bg:#fbfbfb; --muted:#666; --border:#e2e2e2;
          --shade-0:#e6f4ea; --shade-1:#fff4d6; --shade-2:#ffe7b0; --shade-3:#ffcfa3; --shade-4:#ffb3b3; }
  @media (prefers-color-scheme: dark) {
    :root { --fg:#eee; --bg:#171717; --muted:#999; --border:#333;
            --shade-0:#1f3326; --shade-1:#3a3622; --shade-2:#3a3022; --shade-3:#3a261d; --shade-4:#3a1e1e; }
  }
  body { font: 14px/1.5 -apple-system, system-ui, "Segoe UI", sans-serif; color:var(--fg); background:var(--bg); margin:0; padding:2rem; }
  h1 { margin:0 0 0.25rem; font-size:1.5rem; }
  .totals { font-size:1.2rem; margin-bottom:1.5rem; }
  table { width:100%; border-collapse:collapse; }
  th, td { text-align:left; padding:6px 8px; border-bottom:1px solid var(--border); }
  th { font-size:0.85rem; color:var(--muted); font-weight:500; }
  td.num { text-align:right; font-variant-numeric:tabular-nums; }
  .shade-0 { background:var(--shade-0); } .shade-1 { background:var(--shade-1); }
  .shade-2 { background:var(--shade-2); } .shade-3 { background:var(--shade-3); }
  .shade-4 { background:var(--shade-4); }
</style>
</head><body>
<h1>c3x cost diff</h1>
<p class="totals">{{.Currency}}{{.BaselineTotal}}/mo → {{.Currency}}{{.CurrentTotal}}/mo &nbsp; (Δ {{.TotalDelta}})</p>
<table>
<thead><tr><th>Resource</th><th>Change</th><th class="num">Baseline</th><th class="num">Current</th><th class="num">Δ</th></tr></thead>
<tbody>
{{range .Rows}}
<tr class="{{.Shade}}">
  <td>{{.Label}}</td>
  <td>{{.Change}}</td>
  <td class="num">{{$.Currency}}{{.Baseline}}</td>
  <td class="num">{{$.Currency}}{{.Current}}</td>
  <td class="num">{{.Delta}}</td>
</tr>
{{end}}
</tbody>
</table>
</body></html>
`))
