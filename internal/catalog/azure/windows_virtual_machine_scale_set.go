package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"

	"github.com/shopspring/decimal"
)

type WindowsVirtualMachineScaleSet struct {
	Address                               string
	Region                                string
	SKU                                   string
	LicenseType                           string
	AdditionalCapabilitiesUltraSSDEnabled bool
	IsDevTest                             bool
	OSDiskData                            *ManagedDiskData
	Instances                             *int64       `c3x_usage:"instances"`
	OSDisk                                *OSDiskUsage `c3x_usage:"os_disk"`
}

func (r *WindowsVirtualMachineScaleSet) CoreType() string {
	return "WindowsVirtualMachineScaleSet"
}

func (r *WindowsVirtualMachineScaleSet) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "instances", ValueType: engine.Int64, DefaultValue: 0},
		{
			Key:          "os_disk",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "os_disk", Items: OSDiskUsageSchema},
		},
	}
}

func (r *WindowsVirtualMachineScaleSet) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *WindowsVirtualMachineScaleSet) BuildResource() *engine.Estimate {
	region := r.Region

	instanceType := r.SKU
	licenseType := r.LicenseType

	costComponents := []*engine.LineItem{windowsVirtualMachineCostComponent(region, instanceType, licenseType, nil, r.IsDevTest)}

	if r.AdditionalCapabilitiesUltraSSDEnabled {
		costComponents = append(costComponents, ultraSSDReservationCostComponent(region))
	}

	subResources := make([]*engine.Estimate, 0)

	var monthlyDiskOperations *decimal.Decimal
	if r.OSDisk != nil && r.OSDisk.MonthlyDiskOperations != nil {
		monthlyDiskOperations = decimalPtr(decimal.NewFromInt(*r.OSDisk.MonthlyDiskOperations))
	}

	if r.OSDiskData != nil {
		osDisk := osDiskSubResource(region, r.OSDiskData.DiskType, r.OSDiskData.DiskSizeGB, r.OSDiskData.DiskIOPSReadWrite, r.OSDiskData.DiskMBPSReadWrite, monthlyDiskOperations)
		if osDisk != nil {
			subResources = append(subResources, osDisk)
		}
	}

	instanceCount := decimal.NewFromInt(*r.Instances)
	if r.Instances != nil {
		instanceCount = decimal.NewFromInt(*r.Instances)
	}

	res := &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		SubResources:   subResources,
		UsageSchema:    r.UsageSchema(),
	}

	engine.ScaleQuantities(res, instanceCount)

	return res
}
