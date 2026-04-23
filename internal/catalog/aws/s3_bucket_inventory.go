package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type S3BucketInventory struct {
	Address              string
	Region               string
	MonthlyListedObjects *int64 `c3x_usage:"monthly_listed_objects"`
}

func (r *S3BucketInventory) CoreType() string {
	return "S3BucketInventory"
}

func (r *S3BucketInventory) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_listed_objects", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *S3BucketInventory) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *S3BucketInventory) BuildResource() *engine.Estimate {
	var listedObj *decimal.Decimal
	if r.MonthlyListedObjects != nil {
		listedObj = decimalPtr(decimal.NewFromInt(*r.MonthlyListedObjects))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Objects listed",
				Unit:            "1M objects",
				UnitMultiplier:  decimal.NewFromInt(1000000),
				MonthlyQuantity: listedObj,
				ProductFilter: &engine.ProductSelector{
					VendorName: strPtr("aws"),
					Region:     strPtr(r.Region),
					Service:    strPtr("AmazonS3"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/Inventory-ObjectsListed/")},
					},
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
