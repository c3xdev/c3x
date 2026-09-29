package comment_test

import (
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/comment"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
)

// Every comment layout carries the c3x mark (collapsible layouts) and the
// footer (all layouts), in plain Markdown so every forge renders them.
func TestCommentsAreBranded(t *testing.T) {
	est := domain.Estimate{
		ProjectTotal: decimal.RequireFromString("143.81"),
		Currency:     domain.CurrencyUSD,
		Costs: []domain.Cost{{
			Resource:        domain.Reference{Kind: "aws_instance", Name: "web"},
			MonthlySubtotal: decimal.RequireFromString("143.81"),
			Currency:        domain.CurrencyUSD,
			LineItems: []domain.LineItem{{
				Description: "Instance", Quantity: decimal.NewFromInt(730), Unit: "hours",
				UnitRate: decimal.RequireFromString("0.197"), MonthlyCost: decimal.RequireFromString("143.81"),
			}},
		}},
	}
	diff := domain.ComputeDiff(domain.Estimate{Currency: domain.CurrencyUSD}, est)

	for _, expand := range []bool{false, true} {
		plain, err := comment.FormatComment(est, expand)
		if err != nil {
			t.Fatal(err)
		}
		delta, err := comment.FormatCommentDiff(diff, expand)
		if err != nil {
			t.Fatal(err)
		}
		for name, body := range map[string]string{"estimate": plain, "diff": delta} {
			if !strings.HasSuffix(body, comment.Footer) {
				t.Errorf("%s (expand=%v): footer missing:\n%s", name, expand, body)
			}
			if !expand && !strings.HasPrefix(body, "#### ![c3x](https://c3x.dev/brand/c3x-mark.svg) C3X cost estimate") {
				t.Errorf("%s: heading with the c3x mark missing:\n%s", name, body)
			}
			if strings.Contains(body, "<img") || strings.Contains(body, "<sub>") {
				t.Errorf("%s: branding must be plain Markdown for every forge", name)
			}
		}
	}
	if !strings.Contains(comment.Footer, "https://c3x.dev/?utm_source=pr_comment") {
		t.Error("footer must link to c3x.dev with the pr_comment source")
	}
}
