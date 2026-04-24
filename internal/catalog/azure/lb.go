package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"strings"

	"github.com/shopspring/decimal"
)

type LB struct {
	Address                string
	Region                 string
	SKU                    string
	MonthlyDataProcessedGB *float64 `c3x_usage:"monthly_data_processed_gb"`
}

func (r *LB) CoreType() string {
	return "LB"
}

func (r *LB) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_data_processed_gb", ValueType: engine.Float64, DefaultValue: 0}}
}

func (r *LB) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *LB) BuildResource() *engine.Estimate {
	region := convertRegion(r.Region)

	var costComponents []*engine.LineItem
	sku := "Basic"
	var monthlyDataProcessedGb *decimal.Decimal

	if r.MonthlyDataProcessedGB != nil {
		monthlyDataProcessedGb = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataProcessedGB))
	}

	if r.SKU != "" {
		sku = r.SKU
	}

	if strings.ToLower(sku) == "basic" {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	costComponents = append(costComponents, lbDataProcessedCostComponent(region, monthlyDataProcessedGb))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func lbDataProcessedCostComponent(region string, quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Data processed",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Load Balancer"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr("Standard")},
				{Key: "meterName", ValueRegex: regexPtr("Data Processed$")},
			},
		},
		UsageBased: true,
	}
}
