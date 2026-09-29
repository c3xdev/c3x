package calculator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/calculator"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/shopspring/decimal"
)

// scripted answers every lookup with the same rate and source string, so
// each test controls exactly which marker the pricing layer reports.
type scripted struct {
	rate decimal.Decimal
	src  string
}

func (s scripted) Lookup(context.Context, pricing.Query) (decimal.Decimal, string, error) {
	return s.rate, s.src, nil
}

func estimateWith(t *testing.T, src pricing.Source, r domain.Resource) domain.Cost {
	t.Helper()
	reg, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	e := calculator.New(calculator.Options{Registry: reg, Prices: src, Currency: domain.CurrencyUSD, DefaultRegion: "us-east-1"})
	est, err := e.Estimate(context.Background(), []domain.Resource{r})
	if err != nil {
		t.Fatal(err)
	}
	if len(est.Costs) != 1 {
		t.Fatalf("got %d costs", len(est.Costs))
	}
	return est.Costs[0]
}

func codes(c domain.Cost) map[string]int {
	out := map[string]int{}
	for _, cv := range c.Caveats() {
		out[cv.Code]++
	}
	return out
}

func region(s string) *string { return &s }

func TestRegionFallbackIsReported(t *testing.T) {
	t.Parallel()
	c := estimateWith(t, scripted{decimal.RequireFromString("0.096"), "live;fallback=us-east-1"},
		domain.Resource{
			Ref: domain.Reference{Kind: "aws_instance", Name: "web"}, Region: region("sa-east-1"),
			Attributes: map[string]any{"instance_type": "m5.large"},
		})
	if codes(c)[domain.CaveatRegionFallback] == 0 {
		t.Fatalf("want region_fallback, got %v", c.Caveats())
	}
	if d := c.Caveats()[0].Detail; !strings.Contains(d, "sa-east-1") || !strings.Contains(d, "us-east-1") {
		t.Errorf("detail should name both regions: %q", d)
	}
}

func TestCleanLivePriceHasNoCaveats(t *testing.T) {
	t.Parallel()
	c := estimateWith(t, scripted{decimal.RequireFromString("0.096"), domain.PriceSourceLive},
		domain.Resource{
			Ref:        domain.Reference{Kind: "aws_instance", Name: "web"},
			Attributes: map[string]any{"instance_type": "m5.large"},
		})
	if n := len(c.Caveats()); n != 0 {
		t.Errorf("a matched live price must carry no caveats, got %v", c.Caveats())
	}
	if c.LineItems[0].PriceSource != domain.PriceSourceLive {
		t.Errorf("price source = %q", c.LineItems[0].PriceSource)
	}
}

func TestNoMatchIsReported(t *testing.T) {
	t.Parallel()
	c := estimateWith(t, scripted{decimal.Zero, "live;nomatch"},
		domain.Resource{
			Ref:        domain.Reference{Kind: "aws_instance", Name: "web"},
			Attributes: map[string]any{"instance_type": "m5.large"},
		})
	if codes(c)[domain.CaveatNoPrice] == 0 {
		t.Fatalf("a $0 line from a lookup that matched nothing must say so, got %v", c.Caveats())
	}
}

func TestMissingUsageIsReportedOnlyWhenMissing(t *testing.T) {
	t.Parallel()
	nat := func(attrs map[string]any) domain.Resource {
		return domain.Resource{Ref: domain.Reference{Kind: "aws_nat_gateway", Name: "nat"}, Attributes: attrs}
	}
	src := scripted{decimal.RequireFromString("0.045"), domain.PriceSourceLive}
	without := estimateWith(t, src, nat(map[string]any{}))
	if codes(without)[domain.CaveatUsageMissing] == 0 {
		t.Fatalf("NAT data processing with no usage must be reported, got %v", without.Caveats())
	}
	with := estimateWith(t, src, nat(map[string]any{"monthly_data_processed_gb": 100.0}))
	if n := codes(with)[domain.CaveatUsageMissing]; n != 0 {
		t.Errorf("usage supplied, yet %d usage caveat(s): %v", n, with.Caveats())
	}
}

func TestStalePriceIsReported(t *testing.T) {
	t.Parallel()
	c := estimateWith(t, scripted{decimal.RequireFromString("0.096"), "live;stale=240h0m0s"},
		domain.Resource{
			Ref:        domain.Reference{Kind: "aws_instance", Name: "web"},
			Attributes: map[string]any{"instance_type": "m5.large"},
		})
	if codes(c)[domain.CaveatStalePrice] == 0 {
		t.Fatalf("want stale_price, got %v", c.Caveats())
	}
}

// A stub price used to be labelled "live".
func TestStubPriceIsLabelledStub(t *testing.T) {
	t.Parallel()
	c := estimateWith(t, scripted{decimal.Zero, domain.PriceSourceStub},
		domain.Resource{
			Ref:        domain.Reference{Kind: "aws_instance", Name: "web"},
			Attributes: map[string]any{"instance_type": "m5.large"},
		})
	if c.LineItems[0].PriceSource != domain.PriceSourceStub {
		t.Errorf("price source = %q, want stub", c.LineItems[0].PriceSource)
	}
	if codes(c)[domain.CaveatStub] == 0 {
		t.Errorf("want offline_stub caveat, got %v", c.Caveats())
	}
}

// An unresolved attribute is reported only when the kind's pricing reads it.
func TestUnresolvedAttributeReportedOnlyWhenPriceDependsOnIt(t *testing.T) {
	t.Parallel()
	src := scripted{decimal.RequireFromString("0.0104"), domain.PriceSourceLive}
	r := domain.Resource{
		Ref:        domain.Reference{Kind: "aws_instance", Name: "follower"},
		Attributes: map[string]any{"instance_type": nil, "tags": nil},
		Unresolved: []string{"instance_type", "tags"},
	}
	c := estimateWith(t, src, r)
	if n := codes(c)[domain.CaveatUnresolved]; n != 1 {
		t.Fatalf("want exactly 1 unresolved caveat (instance_type, not tags), got %v", c.Caveats())
	}
	if !strings.Contains(c.ResourceCaveats[0].Detail, "instance_type") {
		t.Errorf("detail = %q", c.ResourceCaveats[0].Detail)
	}
}

// An instance count assumed from a data source placeholder is a resource
// caveat naming the assumption, but only on a resource that costs
// something, and never on one whose count assumed nothing.
func TestAssumedCountCaveat(t *testing.T) {
	t.Parallel()
	const assumed = "data.aws_availability_zones.available.names = [us-east-1a, us-east-1b, us-east-1c]"
	priced := scripted{decimal.RequireFromString("0.045"), domain.PriceSourceLive}

	c := estimateWith(t, priced, domain.Resource{
		Ref:          domain.Reference{Kind: "aws_nat_gateway", Name: "nat[0]"},
		Attributes:   map[string]any{},
		AssumedCount: assumed,
	})
	if n := codes(c)[domain.CaveatAssumedCount]; n != 1 {
		t.Fatalf("want 1 assumed_count caveat, got %v", c.Caveats())
	}
	want := "instance count assumes " + assumed + "; price a plan JSON for exact counts"
	if c.ResourceCaveats[0].Detail != want {
		t.Errorf("detail = %q, want %q", c.ResourceCaveats[0].Detail, want)
	}

	free := estimateWith(t, scripted{decimal.Zero, domain.PriceSourceLive}, domain.Resource{
		Ref:          domain.Reference{Kind: "aws_vpc", Name: "main[0]"},
		Attributes:   map[string]any{"cidr_block": "10.0.0.0/16"},
		AssumedCount: assumed,
	})
	if n := codes(free)[domain.CaveatAssumedCount]; n != 0 {
		t.Errorf("a free resource got an assumed_count caveat: %v", free.Caveats())
	}

	literal := estimateWith(t, priced, domain.Resource{
		Ref:        domain.Reference{Kind: "aws_nat_gateway", Name: "nat[0]"},
		Attributes: map[string]any{},
	})
	if n := codes(literal)[domain.CaveatAssumedCount]; n != 0 {
		t.Errorf("a resource with no assumed count got the caveat: %v", literal.Caveats())
	}
}
