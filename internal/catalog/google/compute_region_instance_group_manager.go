package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

// ComputeRegionInstanceGroupManager struct represents Compute Region Instance
// Group Manager resource.
type ComputeRegionInstanceGroupManager struct {
	Address string
	Region  string

	MachineType       string
	PurchaseOption    string
	TargetSize        int64
	ScratchDisks      int
	Disks             []*ComputeDisk
	GuestAccelerators []*ComputeGuestAccelerator
}

func (r *ComputeRegionInstanceGroupManager) CoreType() string {
	return "ComputeRegionInstanceGroupManager"
}

// UsageSchema defines a list which represents the usage schema of ComputeRegionInstanceGroupManager.
func (r *ComputeRegionInstanceGroupManager) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ComputeRegionInstanceGroupManager.
// It uses the `c3x_usage` struct tags to populate data into the ComputeRegionInstanceGroupManager.
func (r *ComputeRegionInstanceGroupManager) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid ComputeRegionInstanceGroupManager struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ComputeRegionInstanceGroupManager) BuildResource() *engine.Estimate {
	costComponents, err := computeCostComponents(r.Region, r.MachineType, r.PurchaseOption, r.TargetSize, nil)
	if err != nil {
		logging.Logger.Warn().Msgf("Skipping resource %s. %s", r.Address, err)
		return nil
	}

	subResources := make([]*engine.Estimate, 0)

	for _, disk := range r.Disks {
		subResources = append(subResources, disk.BuildResource())
	}

	if r.ScratchDisks > 0 {
		costComponents = append(costComponents, scratchDiskCostComponent(r.Region, r.PurchaseOption, r.ScratchDisks*int(r.TargetSize)))
	}

	for _, guestAccel := range r.GuestAccelerators {
		if component := guestAcceleratorCostComponent(r.Region, r.PurchaseOption, guestAccel.Type, guestAccel.Count, r.TargetSize, nil); component != nil {
			costComponents = append(costComponents, component)
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		SubResources:   subResources,
		CostComponents: costComponents,
	}
}
