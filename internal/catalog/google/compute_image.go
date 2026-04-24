package google

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// ComputeImage struct represents Compute Image resource.
type ComputeImage struct {
	Address     string
	Region      string
	StorageSize float64

	// "usage" args
	StorageGB *float64 `c3x_usage:"storage_gb"`
}

func (r *ComputeImage) CoreType() string {
	return "ComputeImage"
}

// UsageSchema defines a list which represents the usage schema of ComputeImage.
func (r *ComputeImage) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "storage_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ComputeImage.
// It uses the `c3x_usage` struct tags to populate data into the ComputeImage.
func (r *ComputeImage) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid ComputeImage struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ComputeImage) BuildResource() *engine.Estimate {
	storageSize := r.StorageSize
	if r.StorageGB != nil {
		storageSize = *r.StorageGB
	}

	var size *decimal.Decimal
	if storageSize > 0 {
		size = decimalPtr(decimal.NewFromFloat(storageSize))
	}

	costComponents := []*engine.LineItem{
		storageImageCostComponent(r.Region, "Storage Image", size),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}
