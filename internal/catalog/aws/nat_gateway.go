package aws

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type NATGateway struct {
	Address string
	Region  string

	MonthlyDataProcessedGB *float64 `c3x_usage:"monthly_data_processed_gb"`
}

func (a *NATGateway) CoreType() string {
	return "NATGateway"
}

func (a *NATGateway) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_data_processed_gb", DefaultValue: 0.0, ValueType: engine.Float64},
	}
}

func (a *NATGateway) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(a, u)
}

func (a *NATGateway) BuildResource() *engine.Estimate {
	var gbDataProcessed *decimal.Decimal
	if a.MonthlyDataProcessedGB != nil {
		gbDataProcessed = decimalPtr(decimal.NewFromFloat(*a.MonthlyDataProcessedGB))
	}

	return &engine.Estimate{
		Name:        a.Address,
		UsageSchema: a.UsageSchema(),
		CostComponents: []*engine.LineItem{
			{
				Name:           "NAT gateway",
				Unit:           "hours",
				UnitMultiplier: decimal.NewFromInt(1),
				HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(a.Region),
					Service:       strPtr("AmazonEC2"),
					ProductFamily: strPtr("NAT Gateway"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/NatGateway-Hours/")},
					},
				},
			},
			{
				Name:            "Data processed",
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: gbDataProcessed,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(a.Region),
					Service:       strPtr("AmazonEC2"),
					ProductFamily: strPtr("NAT Gateway"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/NatGateway-Bytes/")},
					},
				},
				UsageBased: true,
			},
		},
	}
}
