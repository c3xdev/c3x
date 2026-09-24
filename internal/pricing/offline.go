package pricing

import (
	"context"

	"github.com/shopspring/decimal"
)

// OfflineSource serves prices from a warmed on-disk cache with no
// network access. It is the Source behind `c3x estimate --offline`
// once the cache has been populated by [Sync].
//
// It mirrors [HTTPSource]'s reference-region fallback: when a regional
// key misses, it retries the provider's reference region. That means a
// cache warmed for just the reference region answers queries for every
// region (regional deltas are single-digit percent), while a true
// regional miss degrades to the reference rate rather than a silent $0
// — the worst failure mode for a cost tool. The fallback logic is the
// same one the live path uses, so offline and online agree.
type OfflineSource struct {
	cache *DiskCache
}

// NewOfflineSource wraps a DiskCache for offline reads. The cache
// should be opened with WithTTL(-1) so warmed entries never expire
// while disconnected, and with a [NewStub] inner so misses resolve to
// zero (matching live "no priced products" behaviour) instead of
// hitting the network.
func NewOfflineSource(cache *DiskCache) *OfflineSource {
	return &OfflineSource{cache: cache}
}

// Lookup implements [Source].
func (o *OfflineSource) Lookup(ctx context.Context, q Query) (decimal.Decimal, string, error) {
	rate, src, err := o.cache.Lookup(ctx, q)
	if err != nil || !rate.IsZero() {
		return rate, src, err
	}
	if ref := referenceRegion(q.Provider); ref != "" &&
		q.Region != ref && q.Region != "" && q.Region != "global" {
		fq := q
		fq.Region = ref
		if frate, fsrc, ferr := o.cache.Lookup(ctx, fq); ferr == nil && !frate.IsZero() {
			return frate, SourceWith(BaseSource(fsrc), FlagFallback, ref), nil
		}
	}
	return rate, src, err
}

// Close releases the underlying SQLite handle.
func (o *OfflineSource) Close() error { return o.cache.Close() }
