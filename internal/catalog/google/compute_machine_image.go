package google

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// ComputeMachineImage struct represents Compute Machine Image resource.
type ComputeMachineImage struct {
	Address string
	Region  string

	// "usage" args
	StorageGB *float64 `c3x_usage:"storage_gb"`
}

func (r *ComputeMachineImage) CoreType() string {
	return "ComputeMachineImage"
}

// UsageSchema defines a list which represents the usage schema of ComputeMachineImage.
func (r *ComputeMachineImage) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "storage_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ComputeMachineImage.
// It uses the `c3x_usage` struct tags to populate data into the ComputeMachineImage.
func (r *ComputeMachineImage) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid ComputeMachineImage struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ComputeMachineImage) BuildResource() *engine.Estimate {
	var storageSize *decimal.Decimal
	if r.StorageGB != nil {
		storageSize = decimalPtr(decimal.NewFromFloat(*r.StorageGB))
	}

	costComponents := []*engine.LineItem{
		storageImageCostComponent(r.Region, "Storage Machine Image", storageSize),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}
