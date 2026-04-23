package aws

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// GlobalAccelerator struct represents AWS Global Accelerator service
//
// Resource information: https://aws.amazon.com/global-accelerator
// Pricing information: https://aws.amazon.com/global-accelerator/pricing/
type GlobalAccelerator struct {
	Address string
}

func (r *GlobalAccelerator) CoreType() string {
	return "FSxOpenZFSFileSystem"
}

func (r *GlobalAccelerator) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *GlobalAccelerator) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *GlobalAccelerator) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		{
			Name:           "Fixed fee",
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter: &engine.ProductSelector{
				VendorName: strPtr("aws"),
				Service:    strPtr("AWSGlobalAccelerator"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", Value: strPtr("Global-Accelerator-fixed-fee")},
				},
			},
		},
	}
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
