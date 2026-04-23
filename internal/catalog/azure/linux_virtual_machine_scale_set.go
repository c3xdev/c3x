package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"

	"github.com/shopspring/decimal"
)

type LinuxVirtualMachineScaleSet struct {
	Address         string
	SKU             string
	UltraSSDEnabled bool
	Region          string
	OSDiskData      *ManagedDiskData
	Instances       *int64       `c3x_usage:"instances"`
	OSDisk          *OSDiskUsage `c3x_usage:"os_disk"`
}

func (r *LinuxVirtualMachineScaleSet) CoreType() string {
	return "LinuxVirtualMachineScaleSet"
}

func (r *LinuxVirtualMachineScaleSet) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "instances", ValueType: engine.Int64, DefaultValue: 0},
		{
			Key:          "os_disk",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "os_disk", Items: OSDiskUsageSchema},
		},
	}
}

func (r *LinuxVirtualMachineScaleSet) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *LinuxVirtualMachineScaleSet) BuildResource() *engine.Estimate {

	instanceType := r.SKU

	costComponents := []*engine.LineItem{linuxVirtualMachineCostComponent(r.Region, instanceType, nil)}
	subResources := make([]*engine.Estimate, 0)

	if r.UltraSSDEnabled {
		costComponents = append(costComponents, ultraSSDReservationCostComponent(r.Region))
	}

	var monthlyDiskOperations *decimal.Decimal
	if r.OSDisk != nil && r.OSDisk.MonthlyDiskOperations != nil {
		monthlyDiskOperations = decimalPtr(decimal.NewFromInt(*r.OSDisk.MonthlyDiskOperations))
	}

	if r.OSDiskData != nil {
		osDisk := osDiskSubResource(r.Region, r.OSDiskData.DiskType, r.OSDiskData.DiskSizeGB, r.OSDiskData.DiskIOPSReadWrite, r.OSDiskData.DiskMBPSReadWrite, monthlyDiskOperations)
		if osDisk != nil {
			subResources = append(subResources, osDisk)
		}
	}

	instanceCount := decimal.NewFromInt(*r.Instances)

	res := &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		SubResources:   subResources,
		UsageSchema:    r.UsageSchema(),
	}

	engine.ScaleQuantities(res, instanceCount)

	return res
}
