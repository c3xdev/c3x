package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type SecretsManagerSecret struct {
	Address         string
	Region          string
	MonthlyRequests *int64 `c3x_usage:"monthly_requests"`
}

func (r *SecretsManagerSecret) CoreType() string {
	return "SecretsManagerSecret"
}

func (r *SecretsManagerSecret) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_requests", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *SecretsManagerSecret) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *SecretsManagerSecret) BuildResource() *engine.Estimate {
	var monthlyRequests *decimal.Decimal
	if r.MonthlyRequests != nil {
		monthlyRequests = decimalPtr(decimal.NewFromInt(*r.MonthlyRequests))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Secret",
				Unit:            "months",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSSecretsManager"),
					ProductFamily: strPtr("Secret"),
				},
			},
			{
				Name:            "API requests",
				Unit:            "10k requests",
				UnitMultiplier:  decimal.NewFromInt(10000),
				MonthlyQuantity: monthlyRequests,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSSecretsManager"),
					ProductFamily: strPtr("API Request"),
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
