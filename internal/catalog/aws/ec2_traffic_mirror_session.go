package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type EC2TrafficMirrorSession struct {
	Address string
	Region  string
}

func (r *EC2TrafficMirrorSession) CoreType() string {
	return "EC2TrafficMirrorSession"
}

func (r *EC2TrafficMirrorSession) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *EC2TrafficMirrorSession) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EC2TrafficMirrorSession) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:           "Traffic mirror",
				Unit:           "hours",
				UnitMultiplier: decimal.NewFromInt(1),
				HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName: strPtr("aws"),
					Region:     strPtr(r.Region),
					Service:    strPtr("AmazonVPC"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/ENI-Mirror/")},
					},
				},
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
