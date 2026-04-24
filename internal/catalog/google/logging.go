package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type Logging struct {
	Address              string
	MonthlyLoggingDataGB *float64 `c3x_usage:"monthly_logging_data_gb"`
}

func (r *Logging) CoreType() string {
	return "Logging"
}

func (r *Logging) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_logging_data_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *Logging) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *Logging) BuildResource() *engine.Estimate {
	var loggingDataGB *decimal.Decimal
	if r.MonthlyLoggingDataGB != nil {
		loggingDataGB = decimalPtr(decimal.NewFromFloat(*r.MonthlyLoggingDataGB))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: []*engine.LineItem{r.loggingDataCostComponent(loggingDataGB)},
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *Logging) loggingDataCostComponent(quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Logging data",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr("global"),
			Service:       strPtr("Cloud Logging"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", Value: strPtr("Log Volume")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("50"),
		},
		UsageBased: true,
	}
}
