package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type CloudwatchDashboard struct {
	Address string
}

func (r *CloudwatchDashboard) CoreType() string {
	return "CloudwatchDashboard"
}

func (r *CloudwatchDashboard) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *CloudwatchDashboard) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *CloudwatchDashboard) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Dashboard",
				Unit:            "months",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Service:       strPtr("AmazonCloudWatch"),
					ProductFamily: strPtr("Dashboard"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", Value: strPtr("DashboardsUsageHour")},
					},
				},
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
