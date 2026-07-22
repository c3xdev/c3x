package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Estimate is the full result of running the calculator over a set of
// parsed resources. Renderers operate on this type; persistence (the
// saved baseline for `c3x diff`) marshals it as JSON.
type Estimate struct {
	Costs        []Cost
	ProjectTotal decimal.Decimal
	Currency     Currency
	GeneratedAt  time.Time
	// Skipped is the list of resources the parser detected but the
	// calculator could not price — either the resource Kind has no
	// catalog entry, or every dimension was guarded out by `when`
	// predicates. Populated by the calculator engine; renderers can
	// surface it via `--show-skipped` so users see why their estimate
	// undercounts.
	Skipped []SkippedResource
}

// SkippedResource records one resource the calculator decided not
// to price, along with a human-friendly reason. The reason is
// intentionally a string (not a typed enum) so future engine changes
// can refine it without breaking the JSON contract.
type SkippedResource struct {
	Resource Reference
	Reason   string
}

// NewEstimate builds an Estimate, summing per-Cost subtotals at full
// precision and rounding the ProjectTotal to 2dp at the boundary so the
// number a user sees matches the sum of the displayed line items.
//
// Costs with PlanActionDelete are excluded from ProjectTotal — they
// represent resources scheduled for removal that won't exist post-apply.
// They remain in the Costs slice so the delta renderer can display them
// with a `-` marker and subtract their cost in the DELTA line.
func NewEstimate(costs []Cost, currency Currency, generatedAt time.Time) Estimate {
	total := decimal.Zero
	for _, c := range costs {
		if c.Action == PlanActionDelete {
			continue
		}
		total = total.Add(c.MonthlySubtotal)
	}
	return Estimate{
		Costs:        costs,
		ProjectTotal: total.Round(2),
		Currency:     currency,
		GeneratedAt:  generatedAt,
	}
}

// CostFor returns the Cost matching the given Reference, or nil if no
// such resource was in the estimate. Used by Diff and by renderers that
// want to look up a resource by name without scanning the slice.
func (e Estimate) CostFor(ref Reference) *Cost {
	for i := range e.Costs {
		if e.Costs[i].Resource == ref {
			return &e.Costs[i]
		}
	}
	return nil
}
