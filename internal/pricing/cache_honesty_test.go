package pricing

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// flaky answers with a fixed rate and source until failing is set.
type flaky struct {
	rate    decimal.Decimal
	src     string
	failing bool
	calls   int
}

func (f *flaky) Lookup(context.Context, Query) (decimal.Decimal, string, error) {
	f.calls++
	if f.failing {
		return decimal.Zero, "", errors.New("pricing API unreachable")
	}
	return f.rate, f.src, nil
}

var q = Query{Provider: "aws", Service: "AmazonEC2", Region: "us-east-1"}

func openAt(t *testing.T, path string, inner Source, now *time.Time, opts ...DiskCacheOption) *DiskCache {
	t.Helper()
	opts = append(opts, WithClock(func() time.Time { return *now }))
	c, err := OpenDiskCache(path, inner, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// An outage after the TTL used to fail every lookup although a usable
// price sat on disk. It is now served, marked stale.
func TestExpiredPriceServedStaleWhenUpstreamFails(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	inner := &flaky{rate: decimal.RequireFromString("0.096"), src: "live"}
	c := openAt(t, filepath.Join(t.TempDir(), "c.db"), inner, &now)
	if _, _, err := c.Lookup(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	now = now.Add(DefaultCacheTTL + 72*time.Hour)
	inner.failing = true
	rate, src, err := c.Lookup(context.Background(), q)
	if err != nil {
		t.Fatalf("an expired price should be served during an outage, got %v", err)
	}
	if !rate.Equal(decimal.RequireFromString("0.096")) {
		t.Errorf("rate = %s", rate)
	}
	if _, ok := SourceFlag(src, FlagStale); !ok {
		t.Errorf("source %q must be marked stale", src)
	}
}

// With nothing cached, an outage is still an error: there is nothing to serve.
func TestOutageWithNothingCachedIsAnError(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	c := openAt(t, filepath.Join(t.TempDir(), "c.db"), &flaky{failing: true}, &now)
	if _, _, err := c.Lookup(context.Background(), q); err == nil {
		t.Fatal("want an error")
	}
}

// A $0 answer is often a failed match; it must not stick for a week.
func TestZeroResultExpiresAfterAnHour(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	inner := &flaky{rate: decimal.Zero, src: "live;nomatch"}
	c := openAt(t, filepath.Join(t.TempDir(), "c.db"), inner, &now)
	_, _, _ = c.Lookup(context.Background(), q)
	now = now.Add(30 * time.Minute)
	_, _, _ = c.Lookup(context.Background(), q)
	if inner.calls != 1 {
		t.Fatalf("within the hour a $0 should be cached; upstream called %d times", inner.calls)
	}
	now = now.Add(time.Hour)
	_, _, _ = c.Lookup(context.Background(), q)
	if inner.calls != 2 {
		t.Errorf("after an hour a $0 must be re-fetched; upstream called %d times", inner.calls)
	}
}

// The region-fallback marker must survive the round trip through the cache.
func TestFallbackMarkerSurvivesCache(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	inner := &flaky{rate: decimal.RequireFromString("0.0225"), src: SourceWith("live", FlagFallback, "us-east-1")}
	c := openAt(t, filepath.Join(t.TempDir(), "c.db"), inner, &now)
	_, _, _ = c.Lookup(context.Background(), q)
	_, src, _ := c.Lookup(context.Background(), q)
	if inner.calls != 1 {
		t.Fatalf("second lookup should hit the cache")
	}
	if ref, ok := SourceFlag(src, FlagFallback); !ok || ref != "us-east-1" {
		t.Errorf("cached source %q lost the fallback marker", src)
	}
}

// An offline cache synced by an earlier version stores unversioned keys.
// It must keep working after upgrade; an online cache must not trust them.
func TestLegacyKeysReadOnlyForOfflineCaches(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	path := filepath.Join(t.TempDir(), "c.db")
	seed := openAt(t, path, &flaky{}, &now)
	seed.write(queryKey(q), decimal.RequireFromString("0.096"), "live", "legacy row")

	offline := openAt(t, path, &flaky{failing: true}, &now, WithTTL(-1))
	if rate, _, err := offline.Lookup(context.Background(), q); err != nil || !rate.Equal(decimal.RequireFromString("0.096")) {
		t.Errorf("offline cache should read the legacy row: rate=%s err=%v", rate, err)
	}
	inner := &flaky{rate: decimal.RequireFromString("0.1"), src: "live"}
	online := openAt(t, path, inner, &now)
	if rate, _, _ := online.Lookup(context.Background(), q); !rate.Equal(decimal.RequireFromString("0.1")) || inner.calls != 1 {
		t.Errorf("online cache must re-fetch instead of trusting a legacy row: rate=%s calls=%d", rate, inner.calls)
	}
}
