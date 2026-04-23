package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"

	"strings"

	"github.com/shopspring/decimal"
)

type VirtualMachineScaleSet struct {
	Address                   string
	Region                    string
	SKUName                   string
	SKUCapacity               int64
	IsWindows                 bool
	IsDevTest                 bool
	LicenseType               string
	StorageProfileOSDiskData  *ManagedDiskData
	StorageProfileOSDisksData []*ManagedDiskData

	Instances              *int64                     `c3x_usage:"instances"`
	StorageProfileOSDisk   *StorageProfileOSDiskUsage `c3x_usage:"storage_profile_os_disk"`
	StorageProfileDataDisk *StorageProfileOSDiskUsage `c3x_usage:"storage_profile_data_disk"`
}

type StorageProfileOSDiskUsage struct {
	MonthlyDiskOperations *int64 `c3x_usage:"monthly_disk_operations"`
}

type StorageProfileDataDiskUsage struct {
	MonthlyDiskOperations *int64 `c3x_usage:"monthly_disk_operations"`
}

var StorageProfileOSDiskUsageSchema = []*engine.ConsumptionField{
	{ValueType: engine.Int64, DefaultValue: 0, Key: "monthly_disk_operations"},
}

var StorageProfileDataDiskUsageSchema = []*engine.ConsumptionField{
	{ValueType: engine.Int64, DefaultValue: 0, Key: "monthly_disk_operations"},
}

func (r *VirtualMachineScaleSet) CoreType() string {
	return "VirtualMachineScaleSet"
}

func (r *VirtualMachineScaleSet) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "instances", ValueType: engine.Int64, DefaultValue: 0},
		{
			Key:          "storage_profile_os_disk",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "storage_profile_os_disk", Items: StorageProfileOSDiskUsageSchema},
		},
		{
			Key:          "storage_profile_data_disk",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "storage_profile_data_disk", Items: StorageProfileDataDiskUsageSchema},
		},
	}
}

func (r *VirtualMachineScaleSet) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *VirtualMachineScaleSet) BuildResource() *engine.Estimate {
	region := r.Region

	costComponents := []*engine.LineItem{}
	subResources := []*engine.Estimate{}

	instanceType := r.SKUName
	capacity := decimal.NewFromInt(r.SKUCapacity)

	if r.Instances != nil {
		capacity = decimal.NewFromInt(*r.Instances)
	}

	os := "Linux"
	if r.IsWindows {
		os = "Windows"
	}

	if strings.ToLower(os) == "linux" {
		costComponents = append(costComponents, linuxVirtualMachineCostComponent(region, instanceType, nil))
	}

	if strings.ToLower(os) == "windows" {
		licenseType := "Windows_Client"
		if r.LicenseType != "" {
			licenseType = r.LicenseType
		}
		costComponents = append(costComponents, windowsVirtualMachineCostComponent(region, instanceType, licenseType, nil, r.IsDevTest))
	}

	res := &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		SubResources:   subResources,
		UsageSchema:    r.UsageSchema(),
	}

	engine.ScaleQuantities(res, capacity)

	var storageOperations *decimal.Decimal
	if r.StorageProfileOSDisk != nil && r.StorageProfileOSDisk.MonthlyDiskOperations != nil {
		storageOperations = decimalPtr(decimal.NewFromInt(*r.StorageProfileOSDisk.MonthlyDiskOperations))
	}
	if r.StorageProfileOSDiskData != nil {
		res.SubResources = append(res.SubResources, legacyOSDiskSubResource(region, r.StorageProfileOSDiskData.DiskType, r.StorageProfileOSDiskData.DiskSizeGB, r.StorageProfileOSDiskData.DiskIOPSReadWrite, r.StorageProfileOSDiskData.DiskMBPSReadWrite, storageOperations))
	}

	if r.StorageProfileDataDisk != nil && r.StorageProfileDataDisk.MonthlyDiskOperations != nil {
		storageOperations = decimalPtr(decimal.NewFromInt(*r.StorageProfileDataDisk.MonthlyDiskOperations))
	}

	for _, s := range r.StorageProfileOSDisksData {
		res.SubResources = append(res.SubResources, &engine.Estimate{
			Name:           "storage_data_disk",
			CostComponents: managedDiskCostComponents(region, s.DiskType, s.DiskSizeGB, s.DiskIOPSReadWrite, s.DiskMBPSReadWrite, storageOperations),
			UsageSchema:    r.UsageSchema(),
		})
	}

	return res
}
