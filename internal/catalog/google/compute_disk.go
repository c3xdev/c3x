package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// ComputeDisk struct represents Compute Disk resource.
type ComputeDisk struct {
	Address       string
	Region        string
	Type          string
	Size          float64
	InstanceCount *int64

	// applicable for pd-extreme and hyperdisk-extreme
	IOPS int64
}

func (r *ComputeDisk) CoreType() string {
	return "ComputeDisk"
}

// UsageSchema defines a list which represents the usage schema of ComputeDisk.
func (r *ComputeDisk) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ComputeDisk.
// It uses the `c3x_usage` struct tags to populate data into the ComputeDisk.
func (r *ComputeDisk) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid ComputeDisk struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ComputeDisk) BuildResource() *engine.Estimate {
	count := int64(1)
	if r.InstanceCount != nil {
		count = *r.InstanceCount
	}

	costComponents := []*engine.LineItem{
		computeDiskCostComponent(r.Region, r.Type, r.Size, count),
	}

	if r.Type == "pd-extreme" || r.Type == "hyperdisk-extreme" {
		costComponents = append(costComponents, computeDiskIOPSCostComponent(r.Region, r.Type, r.Size, 1, r.IOPS))
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}
