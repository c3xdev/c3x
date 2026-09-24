package catalog_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/catalog"
)

// TestEmbeddedCatalogLoads verifies the bundled TOMLs parse, validate,
// and register without errors. If a future edit breaks a TOML, this
// test fails before any user sees it.
func TestEmbeddedCatalogLoads(t *testing.T) {
	reg, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if reg.Len() < 70 {
		t.Errorf("expected at least 70 definitions, got %d", reg.Len())
	}
}

// TestEveryKindIsUniqueAndAddressableByKind catches a future regression
// where two TOMLs accidentally declare the same kind.
func TestEveryKindIsUniqueAndAddressableByKind(t *testing.T) {
	reg, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	seen := map[string]int{}
	for _, k := range reg.Kinds() {
		seen[k]++
		if def := reg.Get(k); def == nil || def.Kind != k {
			t.Errorf("kind %q not retrievable", k)
		}
	}
	for k, n := range seen {
		if n > 1 {
			t.Errorf("kind %q appears %d times", k, n)
		}
	}
}

// TestStaticRateBucketing exercises the literal-vs-price() detector by
// snapshotting which kinds are currently flagged STATIC. The number is
// allowed to shrink (more live-priced is better) but a sudden jump up
// suggests a regression.
func TestStaticRateBucketing(t *testing.T) {
	reg, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var static []string
	for _, k := range reg.Kinds() {
		// Exclude pure free-shell TOMLs (rate="0" quantity="0"); the
		// real regression signal is genuine STATIC kinds — meters
		// inlined because the upstream catalog doesn't expose them.
		if reg.HasStaticRate(k) && !reg.IsFreeShell(k) {
			static = append(static, k)
		}
	}
	sort.Strings(static)
	// Snapshot: ~50 STATIC entries with inline rates as of 2026-06-08.
	// Loose bound — alerts on a sudden jump that suggests upstream
	// regression, not gradual catalog growth.
	// Cap raised 80 → 110 for the 2026-06-10 long-tail tranches:
	// ~25 deliberately-STATIC additions (services not yet in the
	// upstream scrape, each with last_verified enforced by the
	// verifier). The cap still catches accidental price()->literal
	// regressions.
	if len(static) > 110 {
		t.Errorf("STATIC bucket grew to %d kinds, likely regression:\n%s",
			len(static), strings.Join(static, "\n"))
	}
}

// TestProviderIsValidOnEveryDefinition is a smoke test for the
// validator. If a new TOML lands with provider=foo, Load should fail
// before this test even reaches its body.
func TestProviderIsValidOnEveryDefinition(t *testing.T) {
	reg, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, k := range reg.Kinds() {
		def := reg.Get(k)
		switch def.Provider {
		case "aws", "azure", "gcp":
		default:
			t.Errorf("kind %q has unexpected provider %q", k, def.Provider)
		}
	}
}

// A literal zero rate marks a line billed elsewhere (a container sharing
// its database's throughput, a database in an elastic pool). It is not an
// inline price that can go stale, so it must not make a kind STATIC.
func TestZeroRateIsNotStatic(t *testing.T) {
	reg, err := catalog.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, k := range []string{"azurerm_cosmosdb_sql_container", "azurerm_mssql_database"} {
		if reg.HasStaticRate(k) {
			t.Errorf("%s is classified STATIC; its only literal rate is 0", k)
		}
	}
}
