package render

// CSV renderer. Two columns of context (resource, kind) plus one
// row per LineItem so the spreadsheet user can pivot by kind, sum
// per-resource, or filter by source.
//
// Header row matches the order used by `c3x supported-resources
// --format csv` for visual familiarity:
//
//	resource,kind,dimension,quantity,unit,unit_rate,monthly_cost,source

import (
	"bytes"
	"encoding/csv"
	"fmt"

	"github.com/c3xdev/c3x/internal/domain"
)

// RenderCSV produces the line-item-level CSV. One row per LineItem
// plus a trailing row carrying the project total so the file is
// self-contained when downstream tooling sums the cost column.
func RenderCSV(est domain.Estimate) (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{
		"resource", "kind", "dimension", "quantity", "unit",
		"unit_rate", "monthly_cost", "source", "currency",
	}); err != nil {
		return "", fmt.Errorf("csv header: %w", err)
	}
	currency := est.Currency.String()
	for _, c := range est.Costs {
		ref := c.Resource.Label()
		if len(c.LineItems) == 0 {
			// Resource with no line items — emit one row so it isn't
			// invisible in the spreadsheet. Useful for FREE kinds.
			_ = w.Write([]string{
				ref, c.Resource.Kind, "", "", "", "",
				c.MonthlySubtotal.Round(2).String(),
				"", currency,
			})
			continue
		}
		for _, li := range c.LineItems {
			_ = w.Write([]string{
				ref,
				c.Resource.Kind,
				li.Description,
				li.Quantity.String(),
				li.Unit,
				li.UnitRate.String(),
				li.MonthlyCost.Round(2).String(),
				string(li.PriceSource),
				currency,
			})
		}
	}
	// Trailing project total row — `dimension = "TOTAL"` so the
	// reader can filter or sum-check without arithmetic.
	_ = w.Write([]string{
		"PROJECT", "", "TOTAL", "", "", "",
		est.ProjectTotal.Round(2).String(),
		"", currency,
	})
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
