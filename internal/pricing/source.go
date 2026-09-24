// Package pricing resolves per-unit rates for a [Query]. The package
// is structured as a Source interface plus a small set of concrete
// implementations that compose:
//
//	HTTPSource    — talks GraphQL to pricing.c3x.dev (the live backend)
//	DiskCache     — wraps a Source with a SQLite-backed cache + TTL
//	MemoCache     — wraps a Source with an in-process map
//	Stub          — deterministic in-memory Source for tests and --offline
//
// The production chain is MemoCache(DiskCache(HTTPSource)) built by
// [BuildChain]. The calculator only sees the outermost [Source];
// implementations stay swappable behind one interface.
package pricing

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"
)

// Query is the parameter set for one upstream lookup. The calculator
// builds a Query from a catalog PriceMapping by resolving any `expr`
// filters against the active resource's attributes.
type Query struct {
	Provider         string
	Service          string
	ProductFamily    string
	Region           string
	AttributeFilters []KV
	PurchaseOption   string
	Unit             string
}

// KV is one (attribute, value) pair in a Query's filter list. We use a
// flat slice instead of a map so the calculator can preserve insertion
// order — useful for stable cache keys.
type KV struct {
	Key   string
	Value string
}

// Source is the contract every pricing backend satisfies. Lookup
// returns the per-unit rate, a label identifying where the rate came
// from (see [domain.PriceSourceLive] / [domain.PriceSourceStub] /
// [domain.PriceSourceStatic]), and any error.
//
// A nil error with a zero Decimal means "no priced products matched";
// callers treat that as a legitimately-free dimension, not a failure.
type Source interface {
	Lookup(ctx context.Context, q Query) (decimal.Decimal, string, error)
}

// CacheKey produces a stable, opaque identifier for a Query. Two
// queries that differ only by attribute-filter order share the same
// key — the slice is normalised before hashing. The exact format is
// implementation-defined; callers must not depend on it.
func CacheKey(q Query) string {
	return queryKey(q)
}

// queryKey builds a flat colon-separated string. We keep the layout
// deterministic so Get/Put against the disk cache match across c3x
// versions.
//
// Performance note: this is called once per pricing lookup, which for
// a 100-resource estimate happens thousands of times. We pre-size the
// builder + avoid string concatenation in the loop to keep allocs
// flat at one buffer-grow + final string copy.
func queryKey(q Query) string {
	// Sort filters by key so different insertion orders share one key.
	// In-place insertion sort: filter lists are tiny (typically <6
	// entries) so any O(n log n) sort would pay more in overhead.
	sorted := append([]KV(nil), q.AttributeFilters...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].Key < sorted[j-1].Key; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	// Pre-size the builder: 6 fixed columns + (key=value|) per filter.
	const fixedSize = 32 // generous estimate for the six provider/service/etc cells
	size := fixedSize
	for _, f := range sorted {
		size += len(f.Key) + len(f.Value) + 2 // "|" + "="
	}
	var b strings.Builder
	b.Grow(size)
	b.WriteString(q.Provider)
	b.WriteByte('|')
	b.WriteString(q.Service)
	b.WriteByte('|')
	b.WriteString(q.ProductFamily)
	b.WriteByte('|')
	b.WriteString(q.Region)
	b.WriteByte('|')
	b.WriteString(q.PurchaseOption)
	b.WriteByte('|')
	b.WriteString(q.Unit)
	for _, f := range sorted {
		b.WriteByte('|')
		b.WriteString(f.Key)
		b.WriteByte('=')
		b.WriteString(f.Value)
	}
	return b.String()
}

// Source strings may carry markers after the base label, separated by
// ';': "live;nomatch" when a lookup matched no product, and
// "live;fallback=us-east-1" when the rate was quoted from the provider's
// reference region because the requested region had none. They are
// strings so they persist through the disk cache unchanged.
const (
	FlagNoMatch  = "nomatch"
	FlagFallback = "fallback"
	// FlagStale marks a cached price served past its freshness window
	// because the pricing API could not be reached; the value is its age.
	FlagStale = "stale"
)

// SourceWith appends a marker to a source label. value is "" for flags
// without one.
func SourceWith(base, flag, value string) string {
	if value != "" {
		return base + ";" + flag + "=" + value
	}
	return base + ";" + flag
}

// BaseSource strips any markers: "live;fallback=us-east-1" -> "live".
func BaseSource(src string) string {
	base, _, _ := strings.Cut(src, ";")
	return base
}

// SourceFlag reports whether src carries flag, and its value if any.
func SourceFlag(src, flag string) (string, bool) {
	parts := strings.Split(src, ";")
	for _, p := range parts[1:] {
		name, value, _ := strings.Cut(p, "=")
		if name == flag {
			return value, true
		}
	}
	return "", false
}
