// Command verify_catalog is the catalog health + snapshot harness.
// It exercises every kind in the embedded TOML catalog against the
// per-kind `[fixture]` block (attributes + expected_monthly_cost +
// tolerance/exact). Each kind is reported as:
//
//	[OK]      — non-zero estimate via the live pricing endpoint AND
//	            the snapshot match is within tolerance
//	[STATIC]  — non-zero estimate via inline TOML literal AND the
//	            snapshot matches to the cent (`exact = true`)
//	[FREE]    — explicitly-free kind; zero estimate matches the
//	            `expected_monthly_cost = 0, exact = true` snapshot
//	[ZERO]    — no priced line items where the snapshot expected a
//	            non-zero cost. Regression sentinel.
//	[DRIFT]   — non-zero estimate but outside the snapshot tolerance.
//	            Indicates the upstream catalogue rate changed or the
//	            TOML mappings drifted; not necessarily a bug.
//	[NOFIX]   — TOML has no `[fixture]` block. Catalog hygiene gap.
//	[ERR]     — engine returned an error.
//
// Non-zero exit on ZERO, DRIFT (when -strict), NOFIX, or ERR so CI
// catches regressions and missing fixtures.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/c3xdev/c3x/internal/calculator"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/shopspring/decimal"
)

func main() {
	offline := flag.Bool("offline", false, "use the offline stub instead of pricing.c3x.dev")
	verbose := flag.Bool("verbose", false, "print per-line-item detail for ZERO/DRIFT results")
	endpoint := flag.String("endpoint", pricing.DefaultEndpoint, "pricing GraphQL endpoint")
	strict := flag.Bool("strict", false, "fail on DRIFT or NOFIX as well as ZERO/ERR")
	auditFlag := flag.Bool("audit", false, "report price-spread findings (under-specified mappings)")
	auditRatio := flag.Float64("audit-ratio", 1.5, "flag queries whose max/min non-zero price exceeds this ratio")
	flag.Parse()

	reg, err := catalog.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load catalog: %v\n", err)
		os.Exit(2)
	}

	var prices pricing.Source
	var auditor *auditSource
	if *offline {
		prices = pricing.NewStub()
	} else {
		hs := pricing.NewHTTPSource(pricing.WithEndpoint(*endpoint))
		prices = pricing.NewMemoCache(hs)
		if *auditFlag {
			auditor = &auditSource{inner: prices, http: hs, ratio: *auditRatio}
			prices = auditor
		}
	}

	engine := calculator.New(calculator.Options{
		Registry:      reg,
		Prices:        prices,
		Currency:      domain.CurrencyUSD,
		DefaultRegion: "us-east-1",
	})

	kinds := reg.Kinds()
	sort.Strings(kinds)
	fmt.Printf("Verifying %d resource definitions against %s.\n\n",
		len(kinds), describeSource(*offline, *endpoint))

	var counts struct{ ok, static, free, zero, drift, nofix, stale, errs int }
	// No defer: main ends in os.Exit on failure, which skips
	// defers — cancel is called explicitly on both paths instead.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	staleAt := time.Now().AddDate(0, -6, 0) // 6mo cutoff for STATIC freshness

	for _, kind := range kinds {
		def := reg.Get(kind)
		if def == nil {
			fmt.Printf("[ERR]    %-50s no definition (catalog inconsistency)\n", kind)
			counts.errs++
			continue
		}
		if def.Fixture == nil {
			fmt.Printf("[NOFIX]  %-50s TOML has no [fixture] block\n", kind)
			counts.nofix++
			continue
		}

		if auditor != nil {
			auditor.currentKind = kind
		}
		resource := fixtureResource(kind, def.Fixture)
		est, err := engine.Estimate(ctx, []domain.Resource{resource})
		if err != nil {
			fmt.Printf("[ERR]    %-50s %v\n", kind, err)
			counts.errs++
			continue
		}
		if len(est.Costs) == 0 {
			fmt.Printf("[ZERO]   %-50s no estimate row\n", kind)
			counts.zero++
			continue
		}
		cost := est.Costs[0]
		total, _ := cost.MonthlySubtotal.Float64()
		expected := def.Fixture.ExpectedMonthlyCost

		status, line := classify(kind, cost, total, expected, def.Fixture)
		fmt.Println(line)
		switch status {
		case "OK":
			counts.ok++
		case "STATIC":
			counts.static++
			// STATIC entries should be re-verified periodically;
			// emit a follow-up [STALE] line for any whose
			// last_verified field is empty or older than 6 months.
			if isStale(def.Fixture, staleAt) {
				fmt.Printf("[STALE]  %-50s last_verified missing or older than 6 months\n", kind)
				counts.stale++
			}
		case "FREE":
			counts.free++
		case "ZERO":
			counts.zero++
			if *verbose {
				printLineItems(cost.LineItems)
			}
		case "DRIFT":
			counts.drift++
			if *verbose {
				printLineItems(cost.LineItems)
			}
		}
	}

	fmt.Printf("\nSummary: %d live, %d static, %d free, %d zero, %d drift, %d nofix, %d stale, %d errored — total %d\n",
		counts.ok, counts.static, counts.free, counts.zero, counts.drift, counts.nofix, counts.stale, counts.errs,
		counts.ok+counts.static+counts.free+counts.zero+counts.drift+counts.nofix+counts.errs)

	if auditor != nil {
		auditor.report(os.Stdout)
	}

	bad := counts.errs > 0 || counts.zero > 0
	if *strict {
		bad = bad || counts.drift > 0 || counts.nofix > 0 || counts.stale > 0
	}
	cancel()
	if bad {
		os.Exit(1)
	}
}

// auditSource wraps a Source to flag under-specified mappings: when a
// query matches several non-zero prices with a wide spread, the
// max-non-zero picker is probably selecting a premium SKU (provisioned,
// Multi-AZ, IO-optimised) rather than the intended one.
type auditSource struct {
	inner       pricing.Source
	http        *pricing.HTTPSource
	ratio       float64
	currentKind string
	findings    []auditFinding
}

type auditFinding struct {
	kind   string
	query  string
	min    float64
	max    float64
	chosen float64
	count  int
}

func (a *auditSource) Lookup(ctx context.Context, q pricing.Query) (decimal.Decimal, string, error) {
	rate, src, err := a.inner.Lookup(ctx, q)
	if err == nil && a.http != nil {
		if sp, serr := a.http.Spread(ctx, q); serr == nil && sp.Count > 1 {
			mn, _ := sp.MinNonZero.Float64()
			mx, _ := sp.Max.Float64()
			if mn > 0 && mx/mn >= a.ratio {
				ch, _ := rate.Float64()
				a.findings = append(a.findings, auditFinding{
					kind: a.currentKind, query: auditQuerySummary(q),
					min: mn, max: mx, chosen: ch, count: sp.Count,
				})
			}
		}
	}
	return rate, src, err
}

func (a *auditSource) report(w io.Writer) {
	sort.Slice(a.findings, func(i, j int) bool {
		return a.findings[i].max/a.findings[i].min > a.findings[j].max/a.findings[j].min
	})
	fmt.Fprintf(w, "\n=== AUDIT: %d under-specified queries (max/min non-zero ≥ %.2gx) ===\n",
		len(a.findings), a.ratio)
	for _, f := range a.findings {
		mark := ""
		if f.chosen >= f.max-1e-9 {
			mark = "  <- picking MAX"
		}
		fmt.Fprintf(w, "%6.1fx  %-28s %-40s min=%.4f max=%.4f (%d SKUs) chosen=%.4f%s\n",
			f.max/f.min, f.kind, f.query, f.min, f.max, f.count, f.chosen, mark)
	}
}

func auditQuerySummary(q pricing.Query) string {
	s := q.Service
	if q.ProductFamily != "" {
		s += "/" + q.ProductFamily
	}
	return s
}

// isStale reports whether a STATIC fixture's last_verified date is
// either empty or older than the staleAt cutoff. Empty counts as
// stale so contributors must explicitly mark a STATIC rate as
// freshly verified, never inherit "it's probably still right".
func isStale(fix *catalog.Fixture, staleAt time.Time) bool {
	if fix.LastVerified == "" {
		return true
	}
	verified, err := time.Parse("2006-01-02", fix.LastVerified)
	if err != nil {
		// Bad format → treat as stale; a contributor will fix it
		// when the warning surfaces.
		return true
	}
	return verified.Before(staleAt)
}

// classify decides which bucket a verified kind belongs to and
// returns the formatted line and bucket name.
func classify(kind string, cost domain.Cost, total, expected float64, fix *catalog.Fixture) (string, string) {
	// FREE: expected is 0 and exact, computed must be 0.
	if expected == 0 && fix.Exact {
		if total == 0 {
			return "FREE", fmt.Sprintf("[FREE]   %-50s no billable items (snapshot: $0 exact)", kind)
		}
		return "ZERO", fmt.Sprintf("[ZERO]   %-50s expected $0 exact, got $%s",
			kind, cost.MonthlySubtotal)
	}

	// Anything else with zero cost is a regression.
	if total == 0 {
		return "ZERO", fmt.Sprintf("[ZERO]   %-50s no priced line items (snapshot expected $%g)", kind, expected)
	}

	// Snapshot comparison.
	delta := math.Abs(total - expected)
	tolerance := fix.Tolerance
	if fix.Exact {
		tolerance = 0
	} else if tolerance == 0 {
		tolerance = 0.05
	}

	withinBand := false
	if fix.Exact {
		withinBand = delta < 0.01 // to the cent
	} else {
		withinBand = expected == 0 || delta/expected <= tolerance
	}

	bucket := "OK"
	if cost.HasStaticRate() {
		bucket = "STATIC"
	}

	if !withinBand {
		drift := total - expected
		pct := 0.0
		if expected != 0 {
			pct = (drift / expected) * 100
		}
		return "DRIFT", fmt.Sprintf("[DRIFT]  %-50s $%s/mo (snapshot $%g, drift %+.2f%%)",
			kind, cost.MonthlySubtotal, expected, pct)
	}

	tag := bucket
	if bucket == "STATIC" {
		return tag, fmt.Sprintf("[STATIC] %-50s $%s/mo (snapshot $%g, exact)", kind, cost.MonthlySubtotal, expected)
	}
	return tag, fmt.Sprintf("[OK]     %-50s $%s/mo (snapshot $%g, ±%.0f%%)",
		kind, cost.MonthlySubtotal, expected, tolerance*100)
}

func describeSource(offline bool, endpoint string) string {
	if offline {
		return "the offline stub"
	}
	return endpoint
}

func printLineItems(items []domain.LineItem) {
	for _, li := range items {
		fmt.Printf("         · %s: q=%s rate=$%s cost=$%s\n",
			li.Description, li.Quantity, li.UnitRate, li.MonthlyCost)
	}
}

// fixtureResource builds a domain.Resource from a kind's TOML
// fixture. Region defaults to the per-cloud canonical (us-east-1 /
// eastus / us-central1) unless overridden in the fixture itself.
func fixtureResource(kind string, fix *catalog.Fixture) domain.Resource {
	region := fix.Region
	if region == "" {
		region = defaultRegion(kind)
	}
	attrs := fix.Attributes
	if attrs == nil {
		attrs = map[string]any{}
	}
	return domain.Resource{
		Ref:        domain.Reference{Kind: kind, Name: "test"},
		Attributes: attrs,
		Region:     &region,
	}
}

func defaultRegion(kind string) string {
	switch {
	case strings.HasPrefix(kind, "azurerm_"):
		return "eastus"
	case strings.HasPrefix(kind, "google_"):
		return "us-central1"
	default:
		return "us-east-1"
	}
}
