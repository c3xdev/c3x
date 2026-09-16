package pricing

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
)

// MappingShape is the catalog-agnostic description of one priced lookup
// the engine can issue. The command layer derives these from the
// catalog Registry (so this package stays free of a catalog import) and
// hands them to [Sync]. The fields mirror exactly what
// calculator.buildQuery puts into a [Query], so a warmed entry keys
// identically to a live lookup.
type MappingShape struct {
	Provider      string
	Service       string
	ProductFamily string
	// FixedFilters are the mapping's literal (`const`) attribute
	// filters: the engine always queries these exact (key,value) pairs,
	// so a product only matches this mapping when its attribute for
	// each key equals the literal.
	FixedFilters []KV
	// FilterKeys are the attribute keys whose values come from an `expr`
	// — i.e. from the resource at estimate time. The matching value is
	// taken from each enumerated product. Order is irrelevant (queryKey
	// normalises it).
	FilterKeys []string
	// PurchaseOption must already be resolved by the caller via
	// [ResolvePurchaseOption] (mapping value or provider default). The
	// "any"/"" sentinels mean "no purchaseOption filter", matching the
	// live query.
	PurchaseOption string
	Unit           string
	// RegionOverride pins this lookup to one region regardless of the
	// resource's region (e.g. global services). Empty means "use the
	// sync regions".
	RegionOverride string
}

// SyncOptions configures [Sync].
type SyncOptions struct {
	Endpoint  string
	Token     string
	CachePath string
	// Regions to warm for non-region-pinned shapes. Empty means "the
	// provider's reference region only" — enough for full attribute
	// coverage, since OfflineSource falls back to the reference region
	// for any unsynced region.
	Regions []string
	Shapes  []MappingShape
	// Concurrency bounds in-flight (service,region) units. Kept modest
	// by default to stay gentle on the pricing API.
	Concurrency int
	// Progress, if set, is called once per completed unit. Safe to call
	// from multiple goroutines (Sync serialises the calls).
	Progress func(SyncProgress)
	// Client lets tests inject an *http.Client (e.g. httptest). nil uses
	// a pooled default.
	Client *http.Client
}

// SyncProgress reports one completed (service, region) unit.
type SyncProgress struct {
	Provider string
	Service  string
	Region   string
	Entries  int // entries written for this unit
	Done     int // units completed so far
	Total    int // total units
}

// SyncResult summarises a completed sync.
type SyncResult struct {
	Entries  int
	Services int
	Regions  int
	Units    int
}

const defaultSyncConcurrency = 6

// Sync warms the on-disk price cache at opts.CachePath so that
// `c3x estimate --offline` returns real numbers.
//
// For each (provider, service) in the catalog it enumerates every
// product once (paginated), projects each product onto every mapping's
// filter keys to reconstruct the exact engine query, and resolves the
// rate with the same selection rule the live path uses (purchaseOption
// /unit filter, then max non-zero USD across colliding products). The
// resolved (query → rate) pairs are written in batched transactions.
//
// Nothing here re-parses a vendor bulk index; the API has already done
// the hard catalog work. Sync only mirrors the prices the catalog can
// actually query, keyed the way the engine will ask for them.
func Sync(ctx context.Context, opts SyncOptions) (SyncResult, error) {
	if opts.CachePath == "" {
		return SyncResult{}, fmt.Errorf("pricing.Sync: CachePath is required")
	}
	if len(opts.Shapes) == 0 {
		return SyncResult{}, fmt.Errorf("pricing.Sync: no mapping shapes")
	}
	conc := opts.Concurrency
	if conc <= 0 {
		conc = defaultSyncConcurrency
	}

	cache, err := OpenDiskCache(opts.CachePath, NewStub(), WithTTL(-1))
	if err != nil {
		return SyncResult{}, fmt.Errorf("open cache: %w", err)
	}
	defer func() { _ = cache.Close() }()

	units := planUnits(opts.Shapes, opts.Regions)
	if len(units) == 0 {
		return SyncResult{}, fmt.Errorf("pricing.Sync: no work units planned")
	}

	enum := newProductEnumerator(opts.Client, opts.Endpoint, opts.Token)

	var (
		wg        sync.WaitGroup
		writeMu   sync.Mutex // serialises cache writes + progress
		mu        sync.Mutex // guards shared counters/firstErr
		firstErr  error
		totalRows int
		done      int
		services  = map[string]struct{}{}
		regions   = map[string]struct{}{}
	)
	work := make(chan syncUnit)
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	worker := func() {
		defer wg.Done()
		for u := range work {
			entries, err := resolveUnit(cctx, enum, u)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("%s/%s@%s: %w", u.provider, u.service, u.region, err)
					cancel()
				}
				mu.Unlock()
				continue
			}
			writeMu.Lock()
			n, werr := cache.PutBatch(entries)
			done++
			if opts.Progress != nil {
				opts.Progress(SyncProgress{
					Provider: u.provider, Service: u.service, Region: u.region,
					Entries: n, Done: done, Total: len(units),
				})
			}
			writeMu.Unlock()
			mu.Lock()
			if werr != nil && firstErr == nil {
				firstErr = fmt.Errorf("%s/%s@%s: write: %w", u.provider, u.service, u.region, werr)
				cancel()
			}
			totalRows += n
			services[u.provider+"/"+u.service] = struct{}{}
			regions[u.region] = struct{}{}
			mu.Unlock()
		}
	}

	wg.Add(conc)
	for i := 0; i < conc; i++ {
		go worker()
	}
	for _, u := range units {
		select {
		case work <- u:
		case <-cctx.Done():
		}
	}
	close(work)
	wg.Wait()

	if firstErr != nil {
		return SyncResult{}, firstErr
	}
	return SyncResult{
		Entries:  totalRows,
		Services: len(services),
		Regions:  len(regions),
		Units:    len(units),
	}, nil
}

// syncUnit is one (provider, service, region) enumeration paired with
// the shapes whose query region matches.
type syncUnit struct {
	provider string
	service  string
	region   string
	shapes   []MappingShape
}

// planUnits expands shapes into (provider, service, region) work units.
// A region-pinned shape contributes only its pinned region; every other
// shape contributes each sync region for its provider. Shapes are
// grouped per unit so each service/region is enumerated exactly once.
func planUnits(shapes []MappingShape, regions []string) []syncUnit {
	// index[provider/service/region] -> shapes
	type key struct{ prov, svc, region string }
	index := map[key][]MappingShape{}
	order := []key{}

	add := func(k key, s MappingShape) {
		if _, ok := index[k]; !ok {
			order = append(order, k)
		}
		index[k] = append(index[k], s)
	}

	for i := range shapes {
		s := shapes[i]
		if s.Service == "" {
			continue
		}
		if s.RegionOverride != "" {
			add(key{s.Provider, s.Service, s.RegionOverride}, s)
			continue
		}
		for _, r := range syncRegionsFor(s.Provider, regions) {
			add(key{s.Provider, s.Service, r}, s)
		}
	}

	units := make([]syncUnit, 0, len(order))
	for _, k := range order {
		units = append(units, syncUnit{
			provider: k.prov, service: k.svc, region: k.region, shapes: index[k],
		})
	}
	return units
}

// syncRegionsFor returns the regions to warm for a provider: the
// caller's explicit list, or the provider's reference region as the
// default (OfflineSource extends coverage to all regions via fallback).
func syncRegionsFor(provider string, regions []string) []string {
	if len(regions) > 0 {
		return regions
	}
	if ref := referenceRegion(provider); ref != "" {
		return []string{ref}
	}
	return nil
}

// resolveUnit enumerates a unit's products and builds its cache
// entries. Within a unit, queries that collide (multiple products
// projecting to the same key) keep the max non-zero rate — mirroring
// the live path's "max across matching products" rule.
func resolveUnit(ctx context.Context, enum *productEnumerator, u syncUnit) ([]CacheEntry, error) {
	type acc struct {
		q    Query
		rate decimal.Decimal
	}
	byKey := map[string]*acc{}

	err := enum.each(ctx, u.provider, u.service, u.region, func(p enumProduct) error {
		for i := range u.shapes {
			s := &u.shapes[i]
			if s.ProductFamily != "" && p.ProductFamily != s.ProductFamily {
				continue
			}
			// A literal filter the product can't satisfy means the engine
			// query (which pins that literal) would never match it.
			if !matchesFixed(p.Attributes, s.FixedFilters) {
				continue
			}
			dyn, ok := projectFilters(p.Attributes, s.FilterKeys)
			if !ok {
				continue // product lacks a key this shape filters on
			}
			rate := selectRate(p.Prices, s.PurchaseOption, s.Unit)
			if rate.IsZero() {
				continue
			}
			filters := make([]KV, 0, len(s.FixedFilters)+len(dyn))
			filters = append(filters, s.FixedFilters...)
			filters = append(filters, dyn...)
			q := Query{
				Provider:         s.Provider,
				Service:          s.Service,
				ProductFamily:    s.ProductFamily,
				Region:           u.region,
				AttributeFilters: filters,
				PurchaseOption:   s.PurchaseOption,
				Unit:             s.Unit,
			}
			k := queryKey(q)
			if cur, ok := byKey[k]; ok {
				if rate.GreaterThan(cur.rate) {
					cur.rate = rate
				}
			} else {
				byKey[k] = &acc{q: q, rate: rate}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	entries := make([]CacheEntry, 0, len(byKey))
	for _, a := range byKey {
		entries = append(entries, CacheEntry{Query: a.q, Rate: a.rate, Source: domain.PriceSourceLive})
	}
	return entries, nil
}

// matchesFixed reports whether a product satisfies every literal filter
// (its attribute for each key equals the pinned value).
func matchesFixed(attrs map[string]string, fixed []KV) bool {
	for _, f := range fixed {
		if attrs[f.Key] != f.Value {
			return false
		}
	}
	return true
}

// projectFilters extracts (key, value) pairs for the shape's filter
// keys from a product's attributes. Returns ok=false if any key is
// absent — that product can't satisfy the mapping's filter, so the
// engine would never match it either.
func projectFilters(attrs map[string]string, keys []string) ([]KV, bool) {
	out := make([]KV, 0, len(keys))
	for _, k := range keys {
		v, ok := attrs[k]
		if !ok {
			return nil, false
		}
		out = append(out, KV{Key: k, Value: v})
	}
	return out, true
}

// selectRate applies the same selection as the live path: filter a
// product's prices by purchaseOption/unit (the "any"/"" sentinels skip
// the purchaseOption filter), then take the max non-zero USD. Tiered
// products quote several rates; max-non-zero surfaces the first-tier
// on-demand rate a new workload actually pays.
func selectRate(prices []enumPrice, purchaseOption, unit string) decimal.Decimal {
	includePO := purchaseOption != "" && purchaseOption != "any"
	maxNonZero := decimal.Zero
	for _, p := range prices {
		if includePO && p.PurchaseOption != purchaseOption {
			continue
		}
		if unit != "" && p.Unit != unit {
			continue
		}
		if p.USD == "" {
			continue
		}
		d, err := decimal.NewFromString(p.USD)
		if err != nil || d.IsZero() {
			continue
		}
		if d.GreaterThan(maxNonZero) {
			maxNonZero = d
		}
	}
	return maxNonZero
}

// SortShapes returns shapes in a stable order (provider, service,
// productFamily, filter keys). Used so sync planning and tests are
// deterministic.
func SortShapes(shapes []MappingShape) {
	sort.Slice(shapes, func(i, j int) bool {
		a, b := shapes[i], shapes[j]
		if a.Provider != b.Provider {
			return a.Provider < b.Provider
		}
		if a.Service != b.Service {
			return a.Service < b.Service
		}
		return a.ProductFamily < b.ProductFamily
	})
}
