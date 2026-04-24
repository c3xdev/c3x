package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type EC2ClientVPNEndpoint struct {
	Address string
	Region  string
}

func (r *EC2ClientVPNEndpoint) CoreType() string {
	return "EC2ClientVPNEndpoint"
}

func (r *EC2ClientVPNEndpoint) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *EC2ClientVPNEndpoint) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EC2ClientVPNEndpoint) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:           "Connection",
				Unit:           "hours",
				UnitMultiplier: decimal.NewFromInt(1),
				HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName: strPtr("aws"),
					Region:     strPtr(r.Region),
					Service:    strPtr("AmazonVPC"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/ClientVPN-ConnectionHours/")},
					},
				},
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
