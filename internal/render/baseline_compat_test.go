package render_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/shopspring/decimal"
)

// A baseline saved by c3x before 0.3.19 carries the old label in its
// "resource" field. Diffs match on kind and name, so it must still line
// up with a new estimate: no spurious added/removed pairs.
func TestBaselineFromBeforeTerraformAddressesStillDiffs(t *testing.T) {
	old := []byte(`{
	  "costs": [{
	    "resource": "aws_instance.module.tier[\"web\"].this[0]",
	    "kind": "aws_instance",
	    "name": "module.tier[\"web\"].this[0]",
	    "line_items": [],
	    "monthly_subtotal": "75.98",
	    "currency": "USD"
	  }],
	  "project_total": "75.98",
	  "currency": "USD",
	  "generated_at": "2026-09-28T00:00:00Z"
	}`)
	baseline, err := render.DecodeEstimate(old)
	if err != nil {
		t.Fatal(err)
	}
	current := domain.Estimate{
		Currency:     domain.CurrencyUSD,
		ProjectTotal: decimal.RequireFromString("149.57"),
		Costs: []domain.Cost{{
			Resource:        domain.Reference{Kind: "aws_instance", Name: `module.tier["web"].this[0]`},
			MonthlySubtotal: decimal.RequireFromString("149.57"),
			Currency:        domain.CurrencyUSD,
		}},
	}
	d := domain.ComputeDiff(baseline, current)
	if len(d.Resources) != 1 || d.Resources[0].Kind != domain.DeltaModified {
		t.Fatalf("diff = %+v, want one modified resource", d.Resources)
	}
	if got := d.Resources[0].Resource.Label(); got != `module.tier["web"].aws_instance.this[0]` {
		t.Errorf("label = %q, want the Terraform address", got)
	}
}
