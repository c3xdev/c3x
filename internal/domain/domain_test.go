package domain_test

import (
	"testing"
	"time"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
)

func TestParseCurrency(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in      string
		want    domain.Currency
		wantErr bool
	}{
		{"USD", domain.CurrencyUSD, false},
		{"usd", domain.CurrencyUSD, false},
		{"EUR", domain.CurrencyEUR, false},
		{"GBP", domain.CurrencyGBP, false},
		{"jpy", domain.CurrencyJPY, false},
		{"BRL", domain.CurrencyBRL, false},
		{"", domain.CurrencyUnknown, true},
		{"XYZ", domain.CurrencyUnknown, true},
	} {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got, err := domain.ParseCurrency(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseCurrency(%q) err=%v want=%v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Fatalf("ParseCurrency(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestResourceAttrAccessors(t *testing.T) {
	t.Parallel()

	r := domain.Resource{
		Ref: domain.Reference{Kind: "aws_instance", Name: "web"},
		Attributes: map[string]any{
			"instance_type": "m5.xlarge",
			"count":         float64(3),
			"monitored":     true,
		},
	}

	if v, ok := r.AttrString("instance_type"); !ok || v != "m5.xlarge" {
		t.Errorf("AttrString = (%q,%v), want (m5.xlarge,true)", v, ok)
	}
	if _, ok := r.AttrString("count"); ok {
		t.Errorf("AttrString returned true for a numeric value")
	}
	if v, ok := r.AttrInt("count"); !ok || v != 3 {
		t.Errorf("AttrInt = (%d,%v), want (3,true)", v, ok)
	}
	if v, ok := r.AttrBool("monitored"); !ok || !v {
		t.Errorf("AttrBool = (%v,%v), want (true,true)", v, ok)
	}
}

func TestResourceResolveRegion(t *testing.T) {
	t.Parallel()

	specific := "eu-west-1"
	cases := []struct {
		region   *string
		fallback string
		want     string
	}{
		{nil, "us-east-1", "us-east-1"},
		{&specific, "us-east-1", "eu-west-1"},
		{ptr(""), "us-east-1", "us-east-1"},
	}
	for _, tc := range cases {
		got := domain.Resource{Region: tc.region}.ResolveRegion(tc.fallback)
		if got != tc.want {
			t.Errorf("ResolveRegion = %q, want %q (region=%v fallback=%q)",
				got, tc.want, tc.region, tc.fallback)
		}
	}
}

func TestNewEstimateRoundsTotalToTwoDp(t *testing.T) {
	t.Parallel()

	costs := []domain.Cost{
		{
			Resource:        domain.Reference{Kind: "aws_lambda_function", Name: "fn"},
			MonthlySubtotal: decimal.RequireFromString("0.1234"),
		},
		{
			Resource:        domain.Reference{Kind: "aws_lambda_function", Name: "fn2"},
			MonthlySubtotal: decimal.RequireFromString("0.4321"),
		},
	}
	est := domain.NewEstimate(costs, domain.CurrencyUSD, time.Unix(0, 0))
	if want := decimal.RequireFromString("0.56"); !est.ProjectTotal.Equal(want) {
		t.Errorf("ProjectTotal = %s, want %s", est.ProjectTotal, want)
	}
}

func TestComputeDiffClassifiesEveryResource(t *testing.T) {
	t.Parallel()

	baseRef := domain.Reference{Kind: "aws_instance", Name: "kept"}
	gone := domain.Reference{Kind: "aws_instance", Name: "removed"}
	added := domain.Reference{Kind: "aws_instance", Name: "new"}

	baseline := domain.Estimate{
		Costs: []domain.Cost{
			{Resource: baseRef, MonthlySubtotal: dec("10.00")},
			{Resource: gone, MonthlySubtotal: dec("5.00")},
		},
		ProjectTotal: dec("15.00"),
		Currency:     domain.CurrencyUSD,
	}
	current := domain.Estimate{
		Costs: []domain.Cost{
			{Resource: baseRef, MonthlySubtotal: dec("12.00")},
			{Resource: added, MonthlySubtotal: dec("7.00")},
		},
		ProjectTotal: dec("19.00"),
		Currency:     domain.CurrencyUSD,
	}

	d := domain.ComputeDiff(baseline, current)
	if got := len(d.Resources); got != 3 {
		t.Fatalf("expected 3 deltas, got %d", got)
	}
	if !d.TotalDelta.Equal(dec("4.00")) {
		t.Errorf("TotalDelta = %s, want 4.00", d.TotalDelta)
	}
	kinds := map[domain.Reference]domain.DeltaKind{}
	for _, r := range d.Resources {
		kinds[r.Resource] = r.Kind
	}
	if kinds[baseRef] != domain.DeltaModified {
		t.Errorf("kept resource not Modified")
	}
	if kinds[added] != domain.DeltaAdded {
		t.Errorf("new resource not Added")
	}
	if kinds[gone] != domain.DeltaRemoved {
		t.Errorf("removed resource not Removed")
	}
}

func TestCostHasStaticRate(t *testing.T) {
	t.Parallel()

	cost := domain.Cost{LineItems: []domain.LineItem{
		{PriceSource: domain.PriceSourceLive},
		{PriceSource: domain.PriceSourceStatic},
	}}
	if !cost.HasStaticRate() {
		t.Errorf("expected HasStaticRate=true with mixed sources")
	}
	allLive := domain.Cost{LineItems: []domain.LineItem{
		{PriceSource: domain.PriceSourceLive},
		{PriceSource: domain.PriceSourceLive},
	}}
	if allLive.HasStaticRate() {
		t.Errorf("expected HasStaticRate=false with all live sources")
	}
}

func ptr[T any](v T) *T { return &v }

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// TestTerraformAddress covers the address reconstruction `-target` needs.
// The parsers stash the module path in Name with the kind removed, so the
// kind must be spliced in before the final segment; Label() is the
// display form and is deliberately not a valid address for module
// resources.
func TestTerraformAddress(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ref  domain.Reference
		want string
	}{
		{"root", domain.Reference{Kind: "aws_instance", Name: "web"}, "aws_instance.web"},
		{"count index", domain.Reference{Kind: "aws_instance", Name: "web[0]"}, "aws_instance.web[0]"},
		{"module", domain.Reference{Kind: "aws_instance", Name: "module.frontend.web"},
			"module.frontend.aws_instance.web"},
		{"nested modules", domain.Reference{Kind: "aws_instance", Name: "module.a.module.b.web"},
			"module.a.module.b.aws_instance.web"},
		{"module with indexed resource", domain.Reference{Kind: "aws_instance", Name: "module.frontend.web[2]"},
			"module.frontend.aws_instance.web[2]"},
		// A for_each key containing a dot must not be split on that dot.
		{"for_each key with dot", domain.Reference{Kind: "aws_instance", Name: `web["a.b"]`},
			`aws_instance.web["a.b"]`},
		{"module for_each key with dot", domain.Reference{Kind: "aws_instance", Name: `module.a["x.y"].web`},
			`module.a["x.y"].aws_instance.web`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.ref.TerraformAddress(); got != tc.want {
				t.Errorf("TerraformAddress() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTerraformAddressDiffersFromLabelForModules pins the distinction so
// nobody "simplifies" TerraformAddress back into Label.
func TestTerraformAddressDiffersFromLabelForModules(t *testing.T) {
	t.Parallel()
	ref := domain.Reference{Kind: "aws_instance", Name: "module.frontend.web"}
	if ref.Label() == ref.TerraformAddress() {
		t.Fatal("module Label() must not equal TerraformAddress()")
	}
	if ref.Label() != "aws_instance.module.frontend.web" {
		t.Errorf("Label() changed unexpectedly: %q", ref.Label())
	}
}
