package render

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
)

// RenderText formats an Estimate as a terminal-friendly breakdown. The
// layout uses box-drawing characters for visual structure; callers
// targeting plain ASCII should use [FormatJSON] instead.
//
// Static-rate items get a `(static)` annotation so users see at a
// glance which line items don't track upstream price changes.
func RenderText(est domain.Estimate) string {
	return renderTextEstimate(est, false)
}

// RenderTextDelta formats an Estimate showing only resources with plan
// actions (create/update/delete), annotated with +/~/- markers. Resources
// with no-op action are summarized in a footer line. This gives a
// diff-like view from plan JSON without requiring a baseline file.
func RenderTextDelta(est domain.Estimate) string {
	return renderTextEstimate(est, true)
}

func renderTextEstimate(est domain.Estimate, deltaOnly bool) string {
	if len(est.Costs) == 0 {
		return "c3x: no resources to estimate.\n"
	}
	var b strings.Builder
	cur := est.Currency

	header := "estimate"
	if deltaOnly {
		header = "plan changes"
	}
	fmt.Fprintf(&b, "── c3x %s · %s ─────────────────────────────────────────\n\n", header, cur)

	priced := 0
	unchangedCount := 0
	unchangedCost := decimal.Zero
	deltaCost := decimal.Zero

	for _, c := range est.Costs {
		if len(c.LineItems) == 0 {
			continue
		}

		// In non-delta mode, skip deleted resources entirely — they
		// won't exist post-apply and shouldn't appear in the standard view.
		if !deltaOnly && c.Action == domain.PlanActionDelete {
			continue
		}

		// In delta mode, group no-op resources into a summary.
		if deltaOnly && (c.Action == domain.PlanActionNoOp || c.Action == domain.PlanActionNone) {
			unchangedCount++
			unchangedCost = unchangedCost.Add(c.MonthlySubtotal)
			continue
		}

		priced++
		label := c.Resource.Label()
		annot := ""
		if c.HasStaticRate() {
			annot = "  (some line items use static rates)"
		}
		if deltaOnly {
			marker := actionMarker(c.Action)
			if marker != "" {
				fmt.Fprintf(&b, "  %s %s%s\n", marker, label, annot)
			} else {
				fmt.Fprintf(&b, "  %s%s\n", label, annot)
			}
		} else {
			fmt.Fprintf(&b, "  %s%s\n", label, annot)
		}

		// Track delta cost: deletions subtract, creates/updates add.
		if deltaOnly {
			if c.Action == domain.PlanActionDelete {
				deltaCost = deltaCost.Sub(c.MonthlySubtotal)
			} else {
				deltaCost = deltaCost.Add(c.MonthlySubtotal)
			}
		}

		for _, li := range c.LineItems {
			src := ""
			if li.PriceSource == domain.PriceSourceStatic {
				src = " static"
			}
			fmt.Fprintf(&b, "    %s\n", li.Description)
			fmt.Fprintf(&b, "      %s %s × %s%s = %s%s/mo%s\n",
				li.Quantity, li.Unit,
				cur.Symbol(), li.UnitRate,
				cur.Symbol(), li.MonthlyCost,
				src)
		}
		fmt.Fprintf(&b, "    %s subtotal: %s%s/mo\n\n", label, cur.Symbol(), c.MonthlySubtotal)
	}

	if deltaOnly && unchangedCount > 0 {
		fmt.Fprintf(&b, "  ... %d unchanged resources: %s%s/mo\n\n",
			unchangedCount, cur.Symbol(), unchangedCost.Round(2))
	}

	if priced == 0 && !deltaOnly {
		fmt.Fprintf(&b, "  %d resources parsed; none priced.\n", len(est.Costs))
		fmt.Fprintln(&b, "  This usually means --offline, an unknown resource kind,")
		fmt.Fprintln(&b, "  or that pricing.c3x.dev returned no matching products.")
		return b.String()
	}

	if deltaOnly && priced == 0 {
		fmt.Fprintf(&b, "  No cost-affecting changes in this plan.\n")
		fmt.Fprintf(&b, "  %d resources unchanged at %s%s/mo\n",
			unchangedCount, cur.Symbol(), unchangedCost.Round(2))
		return b.String()
	}

	b.WriteString("  ────────────────────────────────────────────────────────────\n")
	if deltaOnly {
		fmt.Fprintf(&b, "  DELTA: %s/mo (changed resources)\n", signedDelta(cur.Symbol(), deltaCost.Round(2)))
		fmt.Fprintf(&b, "  PROJECT TOTAL: %s%s/mo\n", cur.Symbol(), est.ProjectTotal)
	} else {
		fmt.Fprintf(&b, "  PROJECT TOTAL: %s%s/mo\n", cur.Symbol(), est.ProjectTotal)
	}
	return b.String()
}

// actionMarker returns a visual prefix for plan actions.
func actionMarker(a domain.PlanAction) string {
	switch a {
	case domain.PlanActionCreate:
		return "+"
	case domain.PlanActionUpdate:
		return "~"
	case domain.PlanActionDelete:
		return "-"
	default:
		return ""
	}
}

// signedDelta renders a decimal as a signed cost delta string.
func signedDelta(sym string, d decimal.Decimal) string {
	if d.IsZero() {
		return sym + "0.00"
	}
	if d.IsNegative() {
		return "-" + sym + d.Neg().String()
	}
	return "+" + sym + d.String()
}

// RenderTextDiff formats a Diff in the same visual idiom as
// [RenderText] but with +/- markers and a delta column.
func RenderTextDiff(d domain.Diff) string {
	var b strings.Builder
	cur := d.Currency
	fmt.Fprintf(&b, "── c3x diff · %s ─────────────────────────────────────────────\n\n", cur)
	for _, r := range d.Resources {
		marker := deltaMarker(r.Kind)
		fmt.Fprintf(&b, "  %s %s\n", marker, r.Resource.Label())
		fmt.Fprintf(&b, "      baseline: %s%s/mo   current: %s%s/mo   Δ %s%s\n",
			cur.Symbol(), r.Baseline,
			cur.Symbol(), r.Current,
			signed(cur.Symbol(), r.Delta.String()),
			"")
	}
	b.WriteString("\n  ────────────────────────────────────────────────────────────\n")
	fmt.Fprintf(&b, "  TOTAL: %s%s/mo  →  %s%s/mo   (Δ %s)\n",
		cur.Symbol(), d.BaselineTotal,
		cur.Symbol(), d.CurrentTotal,
		signed(cur.Symbol(), d.TotalDelta.String()))
	return b.String()
}

func deltaMarker(k domain.DeltaKind) string {
	switch k {
	case domain.DeltaAdded:
		return "+"
	case domain.DeltaRemoved:
		return "-"
	case domain.DeltaModified:
		return "~"
	default:
		return " "
	}
}

// signed renders a signed delta with the currency symbol attached to
// the numeric portion. Negative numbers already carry their `-` from
// shopspring/decimal so we don't double-print.
func signed(sym, val string) string {
	if strings.HasPrefix(val, "-") {
		return "-" + sym + strings.TrimPrefix(val, "-")
	}
	return "+" + sym + val
}

// signedWithIndicator returns a signed value prefixed with an
// up/down/flat arrow. Used in PR-comment markdown so reviewers see
// the direction of change at a glance. Pure zero (no change)
// renders as a dash.
func signedWithIndicator(sym, val string) string {
	switch {
	case strings.HasPrefix(val, "-"):
		return "🔻 -" + sym + strings.TrimPrefix(val, "-")
	case val == "0" || val == "0.00":
		return "—"
	default:
		return "🔺 +" + sym + val
	}
}
