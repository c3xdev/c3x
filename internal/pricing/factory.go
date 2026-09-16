package pricing

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/c3xdev/c3x/internal/domain"
)

// ChainOptions configures [BuildChain] — the production wiring of
// memo → disk → http. Zero values get sensible defaults:
//
//	Endpoint  → pricing.c3x.dev/graphql
//	CachePath → empty means "callers must supply"; the CLI defaults
//	            it to the platform XDG cache path.
//	TTL       → DefaultCacheTTL (7 days)
//	Offline   → true returns the bare Stub (no network, no disk)
//	NoCache   → true bypasses the disk layer (still memoised)
type ChainOptions struct {
	Endpoint  string
	Token     string
	CachePath string
	TTL       time.Duration
	Offline   bool
	NoCache   bool
	// Currency, when non-USD and non-Unknown, wraps the chain in
	// an FX converter that multiplies returned rates by the
	// USD→Currency rate from Frankfurter (cached 6h).
	Currency domain.Currency
}

// BuildChain assembles the Source the calculator should receive. The
// returned Source is always non-nil; close it via [Source.Close] when
// the chain includes a disk cache (callers may type-assert to
// [io.Closer] or use [TryClose]).
//
// Wiring matrix:
//
//	Offline=true              → Stub (no network, no disk)
//	NoCache=true              → MemoCache(HTTPSource)
//	otherwise (default)       → MemoCache(DiskCache(HTTPSource))
//
// Callers who already have a Source (tests, dev harnesses) can wrap
// it manually instead of using BuildChain.
func BuildChain(opts ChainOptions) (Source, error) {
	if opts.Offline {
		// Offline reads a cache previously warmed by `c3x pricing sync`:
		// no network, no expiry. A missing or unwarmed cache degrades to
		// the bare stub (every rate $0) rather than erroring — the user
		// simply hasn't synced yet, and the run still parses end-to-end.
		if opts.CachePath != "" {
			if disk, err := OpenDiskCache(opts.CachePath, NewStub(), WithTTL(-1)); err == nil {
				return wrapWithFX(NewOfflineSource(disk), opts.Currency), nil
			}
		}
		return wrapWithFX(NewStub(), opts.Currency), nil
	}
	if opts.Endpoint == "" {
		opts.Endpoint = DefaultEndpoint
	}
	if opts.TTL <= 0 {
		opts.TTL = DefaultCacheTTL
	}

	// HTTPSource wrapped with retry: transient failures get up to
	// DefaultRetryPolicy.MaxAttempts tries before bubbling up. Cache
	// hits never reach this layer, so the retry cost is only paid on
	// real upstream-hitting paths.
	live := withRetry(NewHTTPSource(WithEndpoint(opts.Endpoint), WithToken(opts.Token)), DefaultRetryPolicy)

	var base Source
	if opts.NoCache {
		base = NewMemoCache(live)
	} else {
		if opts.CachePath == "" {
			return nil, fmt.Errorf("BuildChain: CachePath is empty (set --no-cache or supply a path)")
		}
		if err := ensureDir(filepath.Dir(opts.CachePath)); err != nil {
			return nil, fmt.Errorf("preparing cache dir: %w", err)
		}
		disk, err := OpenDiskCache(opts.CachePath, live, WithTTL(opts.TTL))
		if err != nil {
			return nil, err
		}
		base = NewMemoCache(disk)
	}
	return wrapWithFX(base, opts.Currency), nil
}

// wrapWithFX adds a USD → target currency conversion layer when the
// caller requested a non-USD output currency. The FX layer is OUTSIDE
// the cache so cached USD rates stay reusable across currency
// changes (a `--currency EUR` run doesn't invalidate cache entries
// for a future `--currency USD` run).
func wrapWithFX(inner Source, target domain.Currency) Source {
	if target == domain.CurrencyUSD || target == domain.CurrencyUnknown {
		return inner
	}
	return NewFXConverter(inner, target)
}

// TryClose closes a Source's underlying handles if it implements a
// Close method. Calculator owners use it during shutdown so the
// SQLite file is flushed cleanly; for in-memory chains it's a no-op.
func TryClose(s Source) error {
	type closer interface{ Close() error }
	if c, ok := unwrap(s).(closer); ok {
		return c.Close()
	}
	return nil
}

// unwrap walks any wrapper layers (*MemoCache, *FXConverter) to find
// the innermost Source. TryClose needs this so wrapping a DiskCache
// doesn't hide its Close method behind a wrapper that lacks one.
func unwrap(s Source) Source {
	for {
		switch t := s.(type) {
		case *MemoCache:
			s = t.inner
		case *FXConverter:
			s = t.inner
		default:
			return s
		}
	}
}
