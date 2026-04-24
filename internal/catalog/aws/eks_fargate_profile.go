package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type EKSFargateProfile struct {
	Address string
	Region  string
}

func (r *EKSFargateProfile) CoreType() string {
	return "EKSFargateProfile"
}

func (r *EKSFargateProfile) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *EKSFargateProfile) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EKSFargateProfile) BuildResource() *engine.Estimate {
	costComponents := make([]*engine.LineItem, 0)
	costComponents = append(costComponents, r.memoryCostComponent())
	costComponents = append(costComponents, r.vcpuCostComponent())

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *EKSFargateProfile) memoryCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           "Per GB per hour",
		Unit:           "GB",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonEKS"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/Fargate-GB-Hours/")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}
}

func (r *EKSFargateProfile) vcpuCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           "Per vCPU per hour",
		Unit:           "CPU",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonEKS"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/Fargate-vCPU-Hours:perCPU/")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}
}
