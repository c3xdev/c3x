package domain

import "github.com/shopspring/decimal"

// DeltaKind classifies how a resource changed between two Estimates.
type DeltaKind int

const (
	// DeltaAdded means the resource exists in the current estimate but
	// not in the baseline.
	DeltaAdded DeltaKind = iota
	// DeltaRemoved means the resource was in the baseline and is gone.
	DeltaRemoved
	// DeltaModified means the resource exists in both with a different
	// subtotal.
	DeltaModified
	// DeltaUnchanged means the resource exists in both with the same
	// subtotal. Renderers may omit these from default output.
	DeltaUnchanged
)

// String renders the kind for the markdown emoji legend, JSON output,
// and log lines.
func (k DeltaKind) String() string {
	switch k {
	case DeltaAdded:
		return "added"
	case DeltaRemoved:
		return "removed"
	case DeltaModified:
		return "modified"
	case DeltaUnchanged:
		return "unchanged"
	default:
		return "unknown"
	}
}

// ResourceDelta is one row of a Diff.
type ResourceDelta struct {
	Resource Reference
	Baseline decimal.Decimal
	Current  decimal.Decimal
	Delta    decimal.Decimal
	Kind     DeltaKind
}

// Diff is the result of comparing two Estimates. ComputeDiff is the only
// supported constructor — clients should not assemble a Diff by hand.
type Diff struct {
	BaselineTotal decimal.Decimal
	CurrentTotal  decimal.Decimal
	TotalDelta    decimal.Decimal
	Resources     []ResourceDelta
	Currency      Currency
	// Caveats are the current estimate's: they qualify the numbers this
	// change leads to, which is what a reviewer is deciding on.
	Caveats []LabeledCaveat
}

// ComputeDiff compares two Estimates and returns a Diff whose Resources
// slice has one entry per resource that was added, removed, modified,
// or unchanged. The order is stable: first the resources from `current`
// in their original order, then the resources only in `baseline`.
func ComputeDiff(baseline, current Estimate) Diff {
	deltas := make([]ResourceDelta, 0, len(current.Costs)+len(baseline.Costs))

	for _, c := range current.Costs {
		var baseAmount decimal.Decimal
		kind := DeltaAdded
		if existing := baseline.CostFor(c.Resource); existing != nil {
			baseAmount = existing.MonthlySubtotal
			if baseAmount.Equal(c.MonthlySubtotal) {
				kind = DeltaUnchanged
			} else {
				kind = DeltaModified
			}
		}
		deltas = append(deltas, ResourceDelta{
			Resource: c.Resource,
			Baseline: baseAmount,
			Current:  c.MonthlySubtotal,
			Delta:    c.MonthlySubtotal.Sub(baseAmount),
			Kind:     kind,
		})
	}

	for _, c := range baseline.Costs {
		if current.CostFor(c.Resource) != nil {
			continue
		}
		deltas = append(deltas, ResourceDelta{
			Resource: c.Resource,
			Baseline: c.MonthlySubtotal,
			Current:  decimal.Zero,
			Delta:    c.MonthlySubtotal.Neg(),
			Kind:     DeltaRemoved,
		})
	}

	return Diff{
		BaselineTotal: baseline.ProjectTotal,
		CurrentTotal:  current.ProjectTotal,
		TotalDelta:    current.ProjectTotal.Sub(baseline.ProjectTotal),
		Resources:     deltas,
		Currency:      current.Currency,
		Caveats:       current.AllCaveats(),
	}
}
