package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type ConfigConfigurationRecorder struct {
	Address                  string
	Region                   string
	MonthlyConfigItems       *int64 `c3x_usage:"monthly_config_items"`
	MonthlyCustomConfigItems *int64 `c3x_usage:"monthly_custom_config_items"`
}

func (r *ConfigConfigurationRecorder) CoreType() string {
	return "ConfigConfigurationRecorder"
}

func (r *ConfigConfigurationRecorder) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_config_items", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_custom_config_items", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *ConfigConfigurationRecorder) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ConfigConfigurationRecorder) BuildResource() *engine.Estimate {
	var monthlyConfigItems *decimal.Decimal
	if r.MonthlyConfigItems != nil {
		monthlyConfigItems = decimalPtr(decimal.NewFromInt(*r.MonthlyConfigItems))
	}

	var monthlyCustomConfigItems *decimal.Decimal
	if r.MonthlyCustomConfigItems != nil {
		monthlyCustomConfigItems = decimalPtr(decimal.NewFromInt(*r.MonthlyCustomConfigItems))
	}

	costComponents := []*engine.LineItem{}

	costComponents = append(costComponents, &engine.LineItem{
		Name:            "Config items",
		Unit:            "records",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyConfigItems,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSConfig"),
			ProductFamily: strPtr("Management Tools - AWS Config"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("ConfigurationItemRecorded$")},
			},
		},
		UsageBased: true,
	})

	costComponents = append(costComponents, &engine.LineItem{
		Name:            "Custom config items",
		Unit:            "records",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyCustomConfigItems,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSConfig"),
			ProductFamily: strPtr("Management Tools - AWS Config"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("CustomConfigItemRecorded$")},
			},
		},
		UsageBased: true,
	})

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
