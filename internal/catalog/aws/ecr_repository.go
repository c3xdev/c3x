package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type ECRRepository struct {
	Address   string
	Region    string
	StorageGB *float64 `c3x_usage:"storage_gb"`
}

func (r *ECRRepository) CoreType() string {
	return "ECRRepository"
}

func (r *ECRRepository) UsageSchema() []*engine.ConsumptionField {
	return ECRRepositoryUsageSchema
}

var ECRRepositoryUsageSchema = []*engine.ConsumptionField{
	{Key: "storage_gb", ValueType: engine.Float64, DefaultValue: 0},
}

func (r *ECRRepository) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ECRRepository) BuildResource() *engine.Estimate {
	var storageSize *decimal.Decimal
	if r.StorageGB != nil {
		storageSize = decimalPtr(decimal.NewFromFloat(*r.StorageGB))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Storage",
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: storageSize,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonECR"),
					ProductFamily: strPtr("EC2 Container Registry"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", Value: strPtr("TimedStorage-ByteHrs")},
					},
				},
				UsageBased: true,
			},
		},
		UsageSchema: ECRRepositoryUsageSchema,
	}
}
