package render_test

import (
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/shopspring/decimal"
)

func topEstimate() domain.Estimate {
	cost := func(kind, name, amount string) domain.Cost {
		return domain.Cost{
			Resource:        domain.Reference{Kind: kind, Name: name},
			MonthlySubtotal: decimal.RequireFromString(amount),
			Currency:        domain.CurrencyUSD,
		}
	}
	return domain.Estimate{
		Currency:     domain.CurrencyUSD,
		ProjectTotal: decimal.RequireFromString("1105.40"),
		Costs: []domain.Cost{
			cost("aws_instance", "web", "285.32"),
			cost("aws_iam_role", "app", "0"), // free: must never be ranked or targeted
			cost("aws_db_instance", "main", "753.00"),
			cost("aws_instance", "module.frontend.worker", "67.08"),
		},
	}
}

func TestTopCostsSortsDescendingAndDropsFree(t *testing.T) {
	got := render.TopCosts(topEstimate(), 0)
	if len(got) != 3 {
		t.Fatalf("expected 3 priced resources (free one dropped), got %d", len(got))
	}
	want := []string{"main", "web", "module.frontend.worker"}
	for i, w := range want {
		if got[i].Resource.Name != w {
			t.Errorf("rank %d = %q, want %q", i+1, got[i].Resource.Name, w)
		}
	}
	for _, c := range got {
		if c.Resource.Kind == "aws_iam_role" {
			t.Error("a zero-cost resource must not appear in the ranking")
		}
	}
}

func TestTopCostsRespectsLimit(t *testing.T) {
	got := render.TopCosts(topEstimate(), 2)
	if len(got) != 2 {
		t.Fatalf("limit 2 should yield 2 resources, got %d", len(got))
	}
	if got[0].Resource.Name != "main" || got[1].Resource.Name != "web" {
		t.Errorf("limit should keep the two most expensive, got %q and %q",
			got[0].Resource.Name, got[1].Resource.Name)
	}
}

// TestRenderTopTargetsEmitsCanonicalAddresses is the point of the whole
// feature: the emitted flags must be addresses `tofu destroy -target=`
// accepts, including for resources inside modules.
func TestRenderTopTargetsEmitsCanonicalAddresses(t *testing.T) {
	out := render.RenderTopTargets(topEstimate(), 0)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	want := []string{
		"-target=aws_db_instance.main",
		"-target=aws_instance.web",
		"-target=module.frontend.aws_instance.worker",
	}
	if len(lines) != len(want) {
		t.Fatalf("expected %d target lines, got %d:\n%s", len(want), len(lines), out)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d = %q, want %q", i+1, lines[i], w)
		}
	}
	// Nothing but flags: the output is meant to be word-split into a
	// destroy invocation.
	for _, l := range lines {
		if !strings.HasPrefix(l, "-target=") {
			t.Errorf("unexpected non-flag line in targets output: %q", l)
		}
	}
	if strings.Contains(out, "aws_instance.module.") {
		t.Error("emitted a Label()-style address; module targets would be rejected by Terraform")
	}
}

func TestRenderTopTextShowsShareAndSummary(t *testing.T) {
	out := render.RenderTopText(topEstimate(), 2)
	for _, want := range []string{
		"aws_db_instance.main",
		"$753.00/mo",
		"68.1%", // 753.00 / 1105.40
		"Showing 2 of 3 priced resources",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderTopHandlesNoPricedResources(t *testing.T) {
	est := domain.Estimate{
		Currency:     domain.CurrencyUSD,
		ProjectTotal: decimal.Zero,
		Costs: []domain.Cost{{
			Resource:        domain.Reference{Kind: "aws_iam_role", Name: "app"},
			MonthlySubtotal: decimal.Zero,
			Currency:        domain.CurrencyUSD,
		}},
	}
	if got := render.RenderTopTargets(est, 5); got != "" {
		t.Errorf("targets output should be empty when nothing is priced, got %q", got)
	}
	if !strings.Contains(render.RenderTopText(est, 5), "No priced resources") {
		t.Error("text output should say there are no priced resources")
	}
	if _, err := render.RenderTopJSON(est, 5); err != nil {
		t.Errorf("JSON render should not error on an empty ranking: %v", err)
	}
}
