package calculator

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/expr"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/shopspring/decimal"
)

// lineRecorder observes every price() call one dimension makes, keeping
// the raw source string so the markers the pricing layer attaches
// (region fallback, no match, stale cache) reach the line item instead of
// being normalised away.
type lineRecorder struct {
	sources []string
}

func (l *lineRecorder) wrap(lookup expr.PriceLookup) expr.PriceLookup {
	return func(name string) (decimal.Decimal, string, error) {
		rate, src, err := lookup(name)
		if err == nil {
			l.sources = append(l.sources, src)
		}
		return rate, src, err
	}
}

// priceSource is the line's label: static when the rate never called
// price(), stub when any lookup was stubbed, otherwise live. It used to
// say "live" for any price() call, stub prices included.
func (l *lineRecorder) priceSource(dim catalog.DimensionSpec) string {
	if !invokesPrice(dim.Rate) {
		return domain.PriceSourceStatic
	}
	for _, s := range l.sources {
		if pricing.BaseSource(s) == domain.PriceSourceStub {
			return domain.PriceSourceStub
		}
	}
	return domain.PriceSourceLive
}

// caveats explains anything about the line that makes it less than a
// matched price for the resource as configured.
func (l *lineRecorder) caveats(dim catalog.DimensionSpec, r domain.Resource, region string, quantity, rate decimal.Decimal) []domain.Caveat {
	var out []domain.Caveat
	seen := map[string]bool{}
	add := func(code, detail string) {
		if !seen[code+detail] {
			seen[code+detail] = true
			out = append(out, domain.Caveat{Code: code, Detail: detail})
		}
	}
	for _, s := range l.sources {
		if ref, ok := pricing.SourceFlag(s, pricing.FlagFallback); ok {
			add(domain.CaveatRegionFallback, fmt.Sprintf("no price found for %s; the %s rate is shown", orDefault(region, "the resource's region"), ref))
		}
		if age, ok := pricing.SourceFlag(s, pricing.FlagStale); ok {
			add(domain.CaveatStalePrice, fmt.Sprintf("the pricing API was unreachable; a cached price %s old is shown", age))
		}
		if _, ok := pricing.SourceFlag(s, pricing.FlagNoMatch); ok && rate.IsZero() {
			add(domain.CaveatNoPrice, "no matching price was found, so this line is $0 although the resource is not free")
		}
		if pricing.BaseSource(s) == domain.PriceSourceStub {
			add(domain.CaveatStub, "offline stub price, not a real rate")
		}
	}
	// A usage-driven line at a real rate but zero quantity: the usage it
	// reads was never supplied.
	if quantity.IsZero() && !rate.IsZero() {
		if missing := missingUsage(dim.Quantity, r); len(missing) > 0 {
			add(domain.CaveatUsageMissing, fmt.Sprintf("no usage provided for %s; set it in a usage file (--usage) to price this line", strings.Join(missing, ", ")))
		}
	}
	return out
}

// missingUsage lists the identifiers a quantity expression reads that the
// resource does not supply.
func missingUsage(quantity string, r domain.Resource) []string {
	ids, err := expr.Identifiers(quantity)
	if err != nil {
		return nil
	}
	var out []string
	for _, id := range ids {
		if v, ok := r.Attributes[id]; !ok || v == nil {
			out = append(out, id)
		}
	}
	return out
}

// kindReads caches, per catalog kind, every identifier its expressions
// read: dimension predicates, quantities and rates, and mapping filters.
type kindReads struct {
	mu    sync.Mutex
	cache map[string]map[string]bool
}

func (k *kindReads) of(def *catalog.Definition) map[string]bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	if got, ok := k.cache[def.Kind]; ok {
		return got
	}
	reads := map[string]bool{}
	collect := func(src string) {
		ids, _ := expr.Identifiers(src)
		for _, id := range ids {
			reads[id] = true
		}
	}
	for _, d := range def.Dimensions {
		collect(d.When)
		collect(d.Quantity)
		collect(d.Rate)
	}
	for _, m := range def.Mappings {
		for _, f := range m.AttributeFilters {
			collect(f.Expr)
		}
	}
	if k.cache == nil {
		k.cache = map[string]map[string]bool{}
	}
	k.cache[def.Kind] = reads
	return reads
}

// unresolvedCaveats reports each unresolved attribute the kind's pricing
// actually reads. A nested path "root_block_device.volume_size" matches
// both how catalogs address it: the block name, or the flattened alias
// root_block_device_volume_size.
func unresolvedCaveats(r domain.Resource, reads map[string]bool) []domain.Caveat {
	var out []domain.Caveat
	paths := append([]string(nil), r.Unresolved...)
	sort.Strings(paths)
	for _, p := range paths {
		root, _, _ := strings.Cut(p, ".")
		if reads[root] || reads[strings.ReplaceAll(p, ".", "_")] {
			out = append(out, domain.Caveat{
				Code:   domain.CaveatUnresolved,
				Detail: fmt.Sprintf("%s could not be evaluated from the configuration, so the catalog default was used", p),
			})
		}
	}
	return out
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
