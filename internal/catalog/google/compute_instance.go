package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

// ComputeInstance struct represents Compute Instance resource.
type ComputeInstance struct {
	Address string
	Region  string

	MachineType       string
	PurchaseOption    string
	Size              int64
	HasBootDisk       bool
	BootDiskSize      float64
	BootDiskType      string
	ScratchDisks      int
	GuestAccelerators []*ComputeGuestAccelerator

	MonthlyHours *float64 `c3x_usage:"monthly_hrs"`
}

func (r *ComputeInstance) CoreType() string {
	return "ComputeInstance"
}

// UsageSchema defines a list which represents the usage schema of ComputeInstance.
func (r *ComputeInstance) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_hrs", DefaultValue: 730, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the ComputeInstance.
// It uses the `c3x_usage` struct tags to populate data into the ComputeInstance.
func (r *ComputeInstance) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid ComputeInstance struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *ComputeInstance) BuildResource() *engine.Estimate {
	costComponents, err := computeCostComponents(r.Region, r.MachineType, r.PurchaseOption, r.Size, r.MonthlyHours)
	if err != nil {
		logging.Logger.Warn().Msgf("Skipping resource %s. %s", r.Address, err)
		return nil
	}

	if r.HasBootDisk {
		costComponents = append(costComponents, bootDiskCostComponent(r.Region, r.BootDiskSize, r.BootDiskType))
	}

	if r.ScratchDisks > 0 {
		costComponents = append(costComponents, scratchDiskCostComponent(r.Region, r.PurchaseOption, r.ScratchDisks))
	}

	for _, guestAccel := range r.GuestAccelerators {
		if component := guestAcceleratorCostComponent(r.Region, r.PurchaseOption, guestAccel.Type, guestAccel.Count, r.Size, r.MonthlyHours); component != nil {
			costComponents = append(costComponents, component)
		}
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}
