package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type ApplicationInsights struct {
	Address               string
	Region                string
	RetentionInDays       int64
	MonthlyDataIngestedGB *float64 `c3x_usage:"monthly_data_ingested_gb"`
}

func (r *ApplicationInsights) CoreType() string {
	return "ApplicationInsights"
}

func (r *ApplicationInsights) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_data_ingested_gb", ValueType: engine.Float64, DefaultValue: 0}}
}

func (r *ApplicationInsights) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ApplicationInsights) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	var dataIngested *decimal.Decimal
	if r.MonthlyDataIngestedGB != nil {
		dataIngested = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataIngestedGB))
	}
	costComponents = append(costComponents, &engine.LineItem{
		Name:            "Data ingested",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: dataIngested,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Application Insights"),
			ProductFamily: strPtr("Management and Governance"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", "Enterprise Overage Data"))},
				{Key: "skuName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", "Enterprise"))},
			},
		},
		UsageBased: true,
	})

	var dataRetentionDays *decimal.Decimal
	if r.RetentionInDays != 0 {
		dataRetentionDays = decimalPtr(decimal.NewFromInt(r.RetentionInDays))

		if dataRetentionDays.GreaterThan(decimal.NewFromInt(90)) && dataIngested != nil {
			days := dataRetentionDays.Sub(decimal.NewFromInt(90)).Div(decimal.NewFromInt(30))
			qty := decimalPtr(dataIngested.Mul(days))
			costComponents = append(costComponents, &engine.LineItem{
				Name:            fmt.Sprintf("Data retention (%s days)", dataRetentionDays.String()),
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: qty,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("azure"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Application Insights"),
					ProductFamily: strPtr("Management and Governance"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", "Data Retention"))},
						{Key: "skuName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", "Enterprise"))},
					},
				},
			})
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
