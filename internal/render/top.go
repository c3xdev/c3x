package render

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
)

// TopCosts returns the priced resources of an Estimate sorted by monthly
// cost, highest first, capped at limit (limit <= 0 means no cap).
//
// Zero-cost resources are excluded. They are free, so they never belong
// in a "most expensive" ranking, and emitting them as destroy targets
// would be noise at best: tearing down an IAM role or a route table
// association saves nothing while still being a destructive action.
//
// The sort is stable, so resources of equal cost keep their parse order
// and repeated runs over unchanged input produce byte-identical output
// (the list is meant to be diffed and consumed by CI).
func TopCosts(est domain.Estimate, limit int) []domain.Cost {
	out := make([]domain.Cost, 0, len(est.Costs))
	for _, c := range est.Costs {
		if c.MonthlySubtotal.IsPositive() {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].MonthlySubtotal.GreaterThan(out[j].MonthlySubtotal)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// RenderTopTargets emits one `-target=<address>` per line, ready to be
// word-split into a `terraform destroy` / `tofu destroy` invocation:
//
//	tofu destroy $(c3x top --limit 5 --format targets)
//
// Nothing but the flags is written, so the output stays pipe-safe. The
// addresses are canonical Terraform addresses (see
// [domain.Reference.TerraformAddress]), which matters for resources
// inside modules.
func RenderTopTargets(est domain.Estimate, limit int) string {
	var b strings.Builder
	for _, c := range TopCosts(est, limit) {
		fmt.Fprintf(&b, "-target=%s\n", c.Resource.TerraformAddress())
	}
	return b.String()
}

// RenderTopText renders the ranking for a terminal: position, address,
// monthly cost, and share of the project total, followed by a summary
// line stating how much of the total the shown resources account for.
func RenderTopText(est domain.Estimate, limit int) string {
	top := TopCosts(est, limit)
	cur := est.Currency
	var b strings.Builder
	fmt.Fprintf(&b, "── c3x top · %s ───────────────────────────────────────\n\n", cur)
	if len(top) == 0 {
		b.WriteString("  No priced resources found.\n")
		return b.String()
	}

	width := 0
	for _, c := range top {
		if n := len(c.Resource.TerraformAddress()); n > width {
			width = n
		}
	}
	shown := decimal.Zero
	for i, c := range top {
		shown = shown.Add(c.MonthlySubtotal)
		fmt.Fprintf(&b, "  %2d. %-*s  %s%s/mo%s\n",
			i+1, width, c.Resource.TerraformAddress(),
			cur.Symbol(), c.MonthlySubtotal.StringFixed(2),
			shareSuffix(c.MonthlySubtotal, est.ProjectTotal))
	}

	priced := len(TopCosts(est, 0))
	fmt.Fprintf(&b, "\n  Showing %d of %d priced resources, %s%s/mo of %s%s/mo total.\n",
		len(top), priced,
		cur.Symbol(), shown.StringFixed(2),
		cur.Symbol(), est.ProjectTotal.StringFixed(2))
	return b.String()
}

// RenderTopJSON marshals the ranking for tooling. Like the other JSON
// renderers, the shape is a public contract.
func RenderTopJSON(est domain.Estimate, limit int) (string, error) {
	top := TopCosts(est, limit)
	shown := decimal.Zero
	rows := make([]topResourceView, 0, len(top))
	for i, c := range top {
		shown = shown.Add(c.MonthlySubtotal)
		rows = append(rows, topResourceView{
			Rank:         i + 1,
			Address:      c.Resource.TerraformAddress(),
			Resource:     c.Resource.Label(),
			Kind:         c.Resource.Kind,
			MonthlyCost:  c.MonthlySubtotal.String(),
			ShareOfTotal: sharePercent(c.MonthlySubtotal, est.ProjectTotal),
		})
	}
	view := topView{
		Currency:     est.Currency.String(),
		ProjectTotal: est.ProjectTotal.String(),
		ShownTotal:   shown.String(),
		PricedCount:  len(TopCosts(est, 0)),
		Resources:    rows,
	}
	b, err := json.MarshalIndent(view, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal top: %w", err)
	}
	return string(b) + "\n", nil
}

type topView struct {
	Currency     string            `json:"currency"`
	ProjectTotal string            `json:"project_total"`
	ShownTotal   string            `json:"shown_total"`
	PricedCount  int               `json:"priced_count"`
	Resources    []topResourceView `json:"resources"`
}

type topResourceView struct {
	Rank         int    `json:"rank"`
	Address      string `json:"address"`
	Resource     string `json:"resource"`
	Kind         string `json:"kind"`
	MonthlyCost  string `json:"monthly_cost"`
	ShareOfTotal string `json:"share_of_total,omitempty"`
}

// sharePercent renders cost as a percentage of total, or "" when there
// is no total to divide by.
func sharePercent(cost, total decimal.Decimal) string {
	if total.IsZero() {
		return ""
	}
	return cost.Div(total).Mul(decimal.NewFromInt(100)).StringFixed(1) + "%"
}

// shareSuffix is sharePercent formatted as a trailing column, empty when
// the share is not computable.
func shareSuffix(cost, total decimal.Decimal) string {
	s := sharePercent(cost, total)
	if s == "" {
		return ""
	}
	return "  " + s
}
