package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type EIP struct {
	Address   string
	Region    string
	Allocated bool
}

func (r *EIP) CoreType() string {
	return "EIP"
}

func (r *EIP) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *EIP) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EIP) BuildResource() *engine.Estimate {
	// The EIP is free if allocated. AWS does this to encourage efficient use of Elastic IPs
	// and discourage users from leaving unused EIPs lying around in their AWS account.
	if r.Allocated {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:           "IP address (if unused)",
				Unit:           "hours",
				UnitMultiplier: decimal.NewFromInt(1),
				HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonEC2"),
					ProductFamily: strPtr("IP Address"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/ElasticIP:IdleAddress/")},
					},
				},
				PriceFilter: &engine.RateSelector{
					StartUsageAmount: strPtr("1"),
				},
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
