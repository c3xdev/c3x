package calculator_test

import (
	"context"
	"testing"
	"time"

	"github.com/c3xdev/c3x/internal/calculator"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/shopspring/decimal"
)

func freezeAt(ts time.Time) func() time.Time { return func() time.Time { return ts } }

// newEngine wires the engine against the embedded catalog and a stub
// price source. Tests seed the stub with specific (mapping → price)
// fixtures.
func newEngine(t testing.TB, prices *pricing.Stub) *calculator.Engine {
	t.Helper()
	reg, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load: %v", err)
	}
	return calculator.New(calculator.Options{
		Registry:      reg,
		Prices:        prices,
		Currency:      domain.CurrencyUSD,
		DefaultRegion: "us-east-1",
		Now:           freezeAt(time.Unix(0, 0).UTC()),
	})
}

// TestEstimateRendersInlineAwsInstance is the Phase 2 smoke test: a
// hand-crafted aws_instance with the canonical attributes flows through
// the engine and returns a non-zero subtotal pulled from the stub.
func TestEstimateRendersInlineAwsInstance(t *testing.T) {
	t.Parallel()

	stub := pricing.NewStub()
	stub.Set(pricing.Query{
		Provider:       "aws",
		Service:        "AmazonEC2",
		ProductFamily:  "Compute Instance",
		Region:         "us-east-1",
		PurchaseOption: "on_demand",
		AttributeFilters: []pricing.KV{
			{Key: "instanceType", Value: "m5.xlarge"},
			{Key: "operatingSystem", Value: "Linux"},
			{Key: "preInstalledSw", Value: "NA"},
			{Key: "tenancy", Value: "Shared"},
			{Key: "capacitystatus", Value: "Used"},
		},
	}, decimal.RequireFromString("0.192"))

	engine := newEngine(t, stub)
	region := "us-east-1"
	res := domain.Resource{
		Ref:    domain.Reference{Kind: "aws_instance", Name: "web"},
		Region: &region,
		Attributes: map[string]any{
			"instance_type": "m5.xlarge",
		},
	}
	est, err := engine.Estimate(context.Background(), []domain.Resource{res})
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if len(est.Costs) != 1 {
		t.Fatalf("expected 1 cost, got %d", len(est.Costs))
	}
	if est.Costs[0].MonthlySubtotal.IsZero() {
		t.Fatalf("expected non-zero subtotal, got 0\nline_items=%+v", est.Costs[0].LineItems)
	}
	want := decimal.RequireFromString("140.16") // 730 × 0.192
	if !est.ProjectTotal.Equal(want) {
		t.Errorf("ProjectTotal = %s, want %s", est.ProjectTotal, want)
	}
}

// TestUnknownResourceKindReturnsEmptyCost mirrors the steady state
// where a parsed resource isn't in the catalog. The engine must not
// error — it must surface a zero-cost row so the user sees "this
// resource exists, c3x doesn't know how to price it yet".
func TestUnknownResourceKindReturnsEmptyCost(t *testing.T) {
	t.Parallel()

	engine := newEngine(t, pricing.NewStub())
	est, err := engine.Estimate(context.Background(), []domain.Resource{
		{Ref: domain.Reference{Kind: "aws_nonexistent_resource", Name: "x"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(est.Costs) != 1 {
		t.Fatalf("expected 1 cost row, got %d", len(est.Costs))
	}
	if !est.Costs[0].MonthlySubtotal.IsZero() {
		t.Errorf("expected zero subtotal for unknown kind, got %s",
			est.Costs[0].MonthlySubtotal)
	}
	if len(est.Costs[0].LineItems) != 0 {
		t.Errorf("expected no line items, got %d", len(est.Costs[0].LineItems))
	}
}

// TestStaticRateProducesStaticLineItem verifies the literal-rate flag
// makes it into the LineItem so the renderer/verifier can distinguish.
func TestStaticRateProducesStaticLineItem(t *testing.T) {
	t.Parallel()

	engine := newEngine(t, pricing.NewStub())
	res := domain.Resource{
		Ref: domain.Reference{Kind: "aws_eip", Name: "main"},
		// aws_eip uses an inline rate of "0.005" — see resources/aws/aws_eip.toml
	}
	est, err := engine.Estimate(context.Background(), []domain.Resource{res})
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if !est.Costs[0].HasStaticRate() {
		t.Errorf("expected HasStaticRate=true on aws_eip")
	}
	if est.Costs[0].MonthlySubtotal.IsZero() {
		t.Errorf("expected non-zero subtotal from inline rate")
	}
}

// TestRegionOverrideInMappingTakesPrecedence checks the `region = "global"`
// override on a mapping (used by CloudFront, GKE, Log Analytics).
func TestRegionOverrideInMappingTakesPrecedence(t *testing.T) {
	t.Parallel()

	// aws_cloudfront_distribution declares region = "global" on every
	// data-transfer mapping. We pre-seed the stub with the global region
	// to confirm the mapping override wins over the engine's default.
	stub := pricing.NewStub()
	stub.Set(pricing.Query{
		Provider:       "aws",
		Service:        "AmazonCloudFront",
		ProductFamily:  "Data Transfer",
		Region:         "global",
		PurchaseOption: "on_demand",
		AttributeFilters: []pricing.KV{
			{Key: "usagetype", Value: "US-DataTransfer-Out-Bytes"},
		},
	}, decimal.RequireFromString("0.085"))

	engine := newEngine(t, stub)
	res := domain.Resource{
		Ref: domain.Reference{Kind: "aws_cloudfront_distribution", Name: "cdn"},
		Attributes: map[string]any{
			"monthly_data_transfer_us_gb": float64(100),
		},
	}
	est, err := engine.Estimate(context.Background(), []domain.Resource{res})
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if est.Costs[0].MonthlySubtotal.IsZero() {
		t.Fatalf("expected priced result; stub may not have matched query\n%+v",
			est.Costs[0])
	}
}

// TestFilterExprCacheDoesNotCollideBetweenMappings is a regression
// test for the program-cache bug found 2026-06-08: two mappings on
// the same kind that filter on the same attribute key with different
// expressions used to share a cache slot, so the second compile()
// lookup returned the first mapping's program.
//
// The fixture exercises aws_elasticache_serverless_cache, which has
// three mappings (data / ecpu / backup) all filtering on `usagetype`
// with distinct expressions. Each must produce its own distinct rate.
func TestFilterExprCacheDoesNotCollideBetweenMappings(t *testing.T) {
	t.Parallel()

	stub := pricing.NewStub()
	base := pricing.Query{
		Provider:       "aws",
		Service:        "AmazonElastiCache",
		ProductFamily:  "ElastiCache Serverless",
		Region:         "us-east-1",
		PurchaseOption: "on_demand",
	}
	dataQ := base
	dataQ.AttributeFilters = []pricing.KV{
		{Key: "cacheEngine", Value: "Redis"},
		{Key: "operation", Value: "CreateServerlessCache"},
		{Key: "usagetype", Value: "USE1-CachedData:Redis"},
	}
	ecpuQ := base
	ecpuQ.AttributeFilters = []pricing.KV{
		{Key: "operation", Value: "CreateServerlessCache"},
		{Key: "usagetype", Value: "USE1-ElastiCacheProcessingUnits:Redis"},
	}
	backupQ := base
	backupQ.AttributeFilters = []pricing.KV{
		{Key: "operation", Value: "CreateServerlessCacheSnapshot"},
		{Key: "usagetype", Value: "USE1-BackupUsage:Redis"},
	}
	stub.Set(dataQ, decimal.RequireFromString("0.125"))
	stub.Set(ecpuQ, decimal.RequireFromString("0.0000000034"))
	stub.Set(backupQ, decimal.RequireFromString("0.085"))

	engine := newEngine(t, stub)
	res := domain.Resource{
		Ref: domain.Reference{Kind: "aws_elasticache_serverless_cache", Name: "cache"},
		Attributes: map[string]any{
			"engine":                        "redis",
			"monthly_data_storage_gb_hours": float64(100),
			"monthly_ecpu":                  float64(1_000_000),
			"monthly_snapshot_gb":           float64(10),
		},
	}
	est, err := engine.Estimate(context.Background(), []domain.Resource{res})
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	// Expected: data=$12.50 + ecpu≈$0.0034 + backup=$0.85 ≈ $13.35.
	// Pre-fix (bug present): all three mappings collide to data's
	// $0.125 rate; ecpu alone would inflate the subtotal to ~$125k.
	got := est.Costs[0].MonthlySubtotal.InexactFloat64()
	if got > 50.0 {
		t.Errorf("subtotal $%.2f is way above expected ~$13.35 — filter cache "+
			"collision regressed (ecpu mapping is reading data's rate)", got)
	}
}

// TestAuroraInstancePricesByStorageType: Aurora publishes two instance
// SKUs per class, and storage_type selects between them. The attribute is
// declared on the parent cluster and copied onto the instance by the
// parser (internal/parser/inherit.go); by the time the calculator sees it,
// it is an ordinary attribute.
func TestAuroraInstancePricesByStorageType(t *testing.T) {
	t.Parallel()

	instanceQuery := func(storage string) pricing.Query {
		return pricing.Query{
			Provider:       "aws",
			Service:        "AmazonRDS",
			ProductFamily:  "Database Instance",
			Region:         "us-east-1",
			PurchaseOption: "on_demand",
			AttributeFilters: []pricing.KV{
				{Key: "instanceType", Value: "db.r6g.large"},
				{Key: "databaseEngine", Value: "Aurora PostgreSQL"},
				{Key: "deploymentOption", Value: "Single-AZ"},
				{Key: "storage", Value: storage},
			},
		}
	}

	cases := []struct {
		name        string
		storageType string
		wantRate    string // per instance-hour
	}{
		{"aurora standard", "", "0.26"},
		{"aurora io-optimized", "aurora-iopt1", "0.338"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stub := pricing.NewStub()
			// Both SKUs are priced, so a wrong filter would still return
			// a rate: the test fails on the value, not on a zero.
			stub.Set(instanceQuery("EBS Only"), decimal.RequireFromString("0.26"))
			stub.Set(instanceQuery("Aurora IO Optimization Mode"), decimal.RequireFromString("0.338"))

			region := "us-east-1"
			attrs := map[string]any{
				"cluster_identifier": "demo",
				"engine":             "aurora-postgresql",
				"instance_class":     "db.r6g.large",
			}
			if tc.storageType != "" {
				attrs["storage_type"] = tc.storageType
			}
			instance := domain.Resource{
				Ref:        domain.Reference{Kind: "aws_rds_cluster_instance", Name: "one"},
				Region:     &region,
				Attributes: attrs,
			}

			engine := newEngine(t, stub)
			est, err := engine.Estimate(context.Background(), []domain.Resource{instance})
			if err != nil {
				t.Fatalf("Estimate: %v", err)
			}

			var got decimal.Decimal
			for _, c := range est.Costs {
				if c.Resource.Kind == "aws_rds_cluster_instance" {
					got = c.MonthlySubtotal
				}
			}
			want := decimal.RequireFromString(tc.wantRate).Mul(decimal.NewFromInt(730))
			if !got.Equal(want) {
				t.Errorf("instance subtotal = %s, want %s (rate %s/hr x 730)", got, want, tc.wantRate)
			}
		})
	}
}

// TestAuroraInstanceWithoutClusterFallsBackToStandard: an instance with no
// matching cluster in the set (a lone resource, or an HCL reference that
// never resolved) must not error or price at zero. It falls back to Aurora
// Standard, the common case.
func TestAuroraInstanceWithoutClusterFallsBackToStandard(t *testing.T) {
	t.Parallel()
	stub := pricing.NewStub()
	stub.Set(pricing.Query{
		Provider: "aws", Service: "AmazonRDS", ProductFamily: "Database Instance",
		Region: "us-east-1", PurchaseOption: "on_demand",
		AttributeFilters: []pricing.KV{
			{Key: "instanceType", Value: "db.r6g.large"},
			{Key: "databaseEngine", Value: "Aurora PostgreSQL"},
			{Key: "deploymentOption", Value: "Single-AZ"},
			{Key: "storage", Value: "EBS Only"},
		},
	}, decimal.RequireFromString("0.26"))

	region := "us-east-1"
	instance := domain.Resource{
		Ref:    domain.Reference{Kind: "aws_rds_cluster_instance", Name: "orphan"},
		Region: &region,
		Attributes: map[string]any{
			"engine":         "aurora-postgresql",
			"instance_class": "db.r6g.large",
		},
	}
	engine := newEngine(t, stub)
	est, err := engine.Estimate(context.Background(), []domain.Resource{instance})
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	want := decimal.RequireFromString("0.26").Mul(decimal.NewFromInt(730))
	if !est.ProjectTotal.Equal(want) {
		t.Errorf("ProjectTotal = %s, want %s", est.ProjectTotal, want)
	}
}
