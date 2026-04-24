package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

// ComputeInstanceGroupManager struct represents Compute Instance Group Manager
// resource.
type ComputeInstanceGroupManager struct {
	Address string
	Region  string

	MachineType       string
	PurchaseOption    string
	TargetSize        int64
	Disks             []*ComputeDisk
	ScratchDisks      int
	GuestAccelerators []*ComputeGuestAccelerator
}

func (r *ComputeInstanceGroupManager) CoreType() string {
	return "ComputeInstanceGroupManager"
}

// UsageSchema defines a list which represents the usage schema of ComputeInstanceGroupManager.
func (r *ComputeInstanceGroupManager) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ComputeInstanceGroupManager.
// It uses the `c3x_usage` struct tags to populate data into the ComputeInstanceGroupManager.
func (r *ComputeInstanceGroupManager) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid ComputeInstanceGroupManager struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ComputeInstanceGroupManager) BuildResource() *engine.Estimate {
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
