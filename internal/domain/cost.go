package domain

import "github.com/shopspring/decimal"

// LineItem is one priced row contributing to a Resource's monthly cost.
//
// Decimal is used for Quantity, UnitRate, and MonthlyCost because the
// pipeline multiplies tiny per-request rates (Lambda invocations at
// $0.0000002 each) by large volumes (10M requests/month); float64 loses
// precision at this scale.
type LineItem struct {
	// Dimension is the catalog-defined identifier (e.g. "compute_hours",
	// "tier_1_requests"). Stable across versions; the rendered label is
	// the user-facing string.
	Dimension string

	// Description is the human-readable label rendered to users.
	Description string

	// Unit is the display unit ("hours", "GB-month", "1M requests").
	Unit string

	// Quantity is the resolved per-month quantity for this dimension.
	Quantity decimal.Decimal

	// UnitRate is the per-unit price returned by the pricing source.
	UnitRate decimal.Decimal

	// MonthlyCost is Quantity * UnitRate, rounded to 4 decimal places
	// internally so micro-priced items accumulate accurately. Renderers
	// round to 2dp for display only.
	MonthlyCost decimal.Decimal

	// PriceSource records where UnitRate came from. "live" for the
	// pricing API, "static" for an inline TOML literal, "stub" for the
	// offline fixtures used in tests. This lets the verifier and the
	// renderer flag static-rate items so users know they won't track
	// upstream price changes.
	PriceSource string
}

// PriceSourceLive identifies a line item priced via the upstream API.
const PriceSourceLive = "live"

// PriceSourceStatic identifies a line item priced via an inline TOML literal.
const PriceSourceStatic = "static"

// PriceSourceStub identifies a line item priced via the offline test stub.
const PriceSourceStub = "stub"

// Cost is the rollup of every LineItem for one Resource.
//
// MonthlySubtotal is recomputed by the calculator after the dimensions
// resolve; renderers MUST NOT recompute it from LineItems because that
// would diverge from the precision-aware calculation.
type Cost struct {
	Resource        Reference
	LineItems       []LineItem
	MonthlySubtotal decimal.Decimal
	Currency        Currency
	// Action carries the Terraform plan action (create/update/no-op)
	// when the estimate was produced from plan JSON. Empty for HCL
	// estimates. Renderers use this to annotate resources with their
	// change type without requiring a separate baseline.
	Action PlanAction
}

// HasStaticRate reports whether any LineItem in this Cost relies on an
// inline TOML literal instead of a live API lookup. Used by the renderer
// to surface "(approximate)" warnings and by the verifier to bucket
// resources distinctly.
func (c Cost) HasStaticRate() bool {
	for _, li := range c.LineItems {
		if li.PriceSource == PriceSourceStatic {
			return true
		}
	}
	return false
}
