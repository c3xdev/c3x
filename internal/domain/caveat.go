package domain

// Caveat marks a cost that rests on something other than a matched
// price for the resource as configured.
//
// A cost tool's failure mode is not an error, it is a confident wrong
// number: a price quoted from another region, a lookup that matched
// nothing and became $0, an attribute that could not be evaluated and
// silently took a catalog default, a usage-driven cost shown as $0
// because nobody said how much is used. Each of those used to render
// exactly like an accurate line. Caveats carry them to the output, and
// `--strict` turns them into a failure.
type Caveat struct {
	// Code is stable and machine-readable; see the Caveat* constants.
	Code string
	// Detail is the human-readable explanation, specific to the line.
	Detail string
}

const (
	// CaveatRegionFallback: no price exists for the resource's region
	// under the catalog's filters, so the provider's reference region
	// (us-east-1, eastus, us-central1) was quoted instead.
	CaveatRegionFallback = "region_fallback"
	// CaveatNoPrice: the price lookup matched no product, so the line
	// is $0 without the resource being free.
	CaveatNoPrice = "no_price"
	// CaveatUsageMissing: the quantity depends on usage (requests, GB
	// transferred, ...) that was not provided, so the line is $0.
	CaveatUsageMissing = "usage_not_provided"
	// CaveatUnresolved: an attribute the price depends on could not be
	// evaluated statically, so a catalog default was used for it.
	CaveatUnresolved = "unresolved_attribute"
	// CaveatStub: priced with the offline stub, not a real price.
	CaveatStub = "offline_stub"
	// CaveatStalePrice: the pricing API was unreachable, so a cached
	// price past its freshness window was used.
	CaveatStalePrice = "stale_price"
)

// Caveats returns every caveat on the cost: its own, then each line
// item's, in order.
func (c Cost) Caveats() []Caveat {
	out := append([]Caveat{}, c.ResourceCaveats...)
	for _, li := range c.LineItems {
		out = append(out, li.Caveats...)
	}
	return out
}

// CaveatCount is the number of caveats across the estimate.
func (e Estimate) CaveatCount() int {
	n := 0
	for _, c := range e.Costs {
		n += len(c.Caveats())
	}
	return n
}

// LabeledCaveat is a caveat with the resource (and line, if any) it
// belongs to, for summaries that list caveats outside the breakdown.
type LabeledCaveat struct {
	Resource Reference
	// Line is the line item's description; empty for a resource caveat.
	Line string
	Caveat
}

// AllCaveats lists every caveat in the estimate, labelled.
func (e Estimate) AllCaveats() []LabeledCaveat {
	var out []LabeledCaveat
	for _, c := range e.Costs {
		for _, cv := range c.ResourceCaveats {
			out = append(out, LabeledCaveat{Resource: c.Resource, Caveat: cv})
		}
		for _, li := range c.LineItems {
			for _, cv := range li.Caveats {
				out = append(out, LabeledCaveat{Resource: c.Resource, Line: li.Description, Caveat: cv})
			}
		}
	}
	return out
}
