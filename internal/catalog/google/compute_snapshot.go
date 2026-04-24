package google

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// ComputeSnapshot struct represents Compute Snapshot resource.
type ComputeSnapshot struct {
	Address  string
	Region   string
	DiskSize float64

	// "usage" args
	StorageGB *float64 `c3x_usage:"storage_gb"`
}

func (r *ComputeSnapshot) CoreType() string {
	return "ComputeSnapshot"
}

// UsageSchema defines a list which represents the usage schema of ComputeSnapshot.
func (r *ComputeSnapshot) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "storage_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ComputeSnapshot.
// It uses the `c3x_usage` struct tags to populate data into the ComputeSnapshot.
func (r *ComputeSnapshot) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid ComputeSnapshot struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ComputeSnapshot) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.storageCostComponent(),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

// storageCostComponent returns a cost component for snapshot storage.
func (r *ComputeSnapshot) storageCostComponent() *engine.LineItem {
	description := "Storage PD Snapshot"

	size := r.DiskSize
	if r.StorageGB != nil {
		size = *r.StorageGB
	}

	var snapshotDiskSize *decimal.Decimal
	if size > 0 {
		snapshotDiskSize = decimalPtr(decimal.NewFromFloat(size))
	}

	return &engine.LineItem{
		Name:            "Storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: snapshotDiskSize,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Compute Engine"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: regexPtr(description)},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("5"),
		},
		UsageBased: true,
	}
}
