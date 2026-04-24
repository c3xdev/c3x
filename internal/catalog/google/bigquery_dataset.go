package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type BigQueryDataset struct {
	Address          string
	Region           string
	MonthlyQueriesTB *float64 `c3x_usage:"monthly_queries_tb"`
}

func (r *BigQueryDataset) CoreType() string {
	return "BigQueryDataset"
}

func (r *BigQueryDataset) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_queries_tb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *BigQueryDataset) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *BigQueryDataset) BuildResource() *engine.Estimate {
	var queriesTB *decimal.Decimal
	if r.MonthlyQueriesTB != nil {
		queriesTB = decimalPtr(decimal.NewFromFloat(*r.MonthlyQueriesTB))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Queries (on-demand)",
				Unit:            "TB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: queriesTB,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr(r.Region),
					Service:       strPtr("BigQuery"),
					ProductFamily: strPtr("ApplicationServices"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", Value: strPtr(fmt.Sprintf("Analysis (%s)", r.Region))},
					},
				},
				PriceFilter: &engine.RateSelector{
					StartUsageAmount: strPtr("1"),
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
