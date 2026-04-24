package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"

	"strings"

	"github.com/shopspring/decimal"
)

type VirtualMachine struct {
	Address                    string
	Region                     string
	StorageImageReferenceOffer string
	VMSize                     string
	StorageOSDiskOSType        string
	LicenseType                string
	StorageOSDiskData          *ManagedDiskData
	OSDiskData                 *ManagedDiskData
	StoragesDiskData           []*ManagedDiskData
	MonthlyHours               *float64              `c3x_usage:"monthly_hrs"`
	StorageOSDisk              *StorageOSDiskUsage   `c3x_usage:"storage_os_disk"`
	StorageDataDisk            *StorageDataDiskUsage `c3x_usage:"storage_data_disk"`
	IsDevTest                  bool
}

type StorageOSDiskUsage struct {
	MonthlyDiskOperations *int64 `c3x_usage:"monthly_disk_operations"`
}

type StorageDataDiskUsage struct {
	MonthlyDiskOperations *int64 `c3x_usage:"monthly_disk_operations"`
}

var StorageOSDiskUsageSchema = []*engine.ConsumptionField{
	{ValueType: engine.Int64, DefaultValue: 0, Key: "monthly_disk_operations"},
}

var StorageDataDiskUsageSchema = []*engine.ConsumptionField{
	{ValueType: engine.Int64, DefaultValue: 0, Key: "monthly_disk_operations"},
}

func (r *VirtualMachine) CoreType() string {
	return "VirtualMachine"
}

func (r *VirtualMachine) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_hrs", ValueType: engine.Float64, DefaultValue: 0},
		{
			Key:          "storage_os_disk",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "storage_os_disk", Items: StorageOSDiskUsageSchema},
		},
		{
			Key:          "storage_data_disk",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "storage_data_disk", Items: StorageDataDiskUsageSchema},
		},
	}
}

func (r *VirtualMachine) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *VirtualMachine) BuildResource() *engine.Estimate {
	region := r.Region

	costComponents := []*engine.LineItem{}
	instanceType := r.VMSize

	os := "Linux"
	if r.StorageImageReferenceOffer != "" {
		if strings.ToLower(r.StorageImageReferenceOffer) == "windowsserver" {
			os = "Windows"
		}
	}
	if strings.ToLower(r.StorageOSDiskOSType) == "windows" {
		os = "Windows"
	}

	if strings.ToLower(os) == "windows" {
		licenseType := r.LicenseType
		costComponents = append(costComponents, windowsVirtualMachineCostComponent(region, instanceType, licenseType, r.MonthlyHours, r.IsDevTest))
	} else {
		costComponents = append(costComponents, linuxVirtualMachineCostComponent(region, instanceType, r.MonthlyHours))
	}

	// Ultra SSD reservation is usage-based (nil quantity) since the legacy azurerm_virtual_machine
	// resource doesn't expose whether ultra SSD is enabled.
	costComponents = append(costComponents, ultraSSDReservationCostComponent(region))

	var storageOperations *decimal.Decimal
	if r.StorageOSDisk != nil && r.StorageOSDisk.MonthlyDiskOperations != nil {
		storageOperations = decimalPtr(decimal.NewFromInt(*r.StorageOSDisk.MonthlyDiskOperations))
	}

	subResources := []*engine.Estimate{}

	if r.StorageOSDiskData != nil {
		subResources = append(subResources, legacyOSDiskSubResource(region, r.StorageOSDiskData.DiskType, r.StorageOSDiskData.DiskSizeGB, r.StorageOSDiskData.DiskIOPSReadWrite, r.StorageOSDiskData.DiskMBPSReadWrite, storageOperations))
	}

	if r.StorageOSDisk != nil && r.StorageDataDisk.MonthlyDiskOperations != nil {
		storageOperations = decimalPtr(decimal.NewFromInt(*r.StorageDataDisk.MonthlyDiskOperations))
	}

	for _, s := range r.StoragesDiskData {
		subResources = append(subResources, &engine.Estimate{
			Name:           "storage_data_disk",
			CostComponents: managedDiskCostComponents(region, s.DiskType, s.DiskSizeGB, s.DiskIOPSReadWrite, s.DiskMBPSReadWrite, storageOperations),
			UsageSchema:    r.UsageSchema(),
		})
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		SubResources:   subResources,
		UsageSchema:    r.UsageSchema(),
	}
}

func ultraSSDReservationCostComponent(region string) *engine.LineItem {
	return &engine.LineItem{
		Name:           "Ultra disk reservation (if unattached)",
		Unit:           "vCPU",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: nil,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Storage"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Ultra Disks")},
				{Key: "skuName", Value: strPtr("Ultra LRS")},
				{Key: "meterName", ValueRegex: regexPtr("Reservation per vCPU Provisioned$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func legacyOSDiskSubResource(region, diskType string, diskSizeGB, diskIOPSReadWrite, diskMBPSReadWrite int64, monthlyDiskOperations *decimal.Decimal) *engine.Estimate {
	return &engine.Estimate{
		Name:           "storage_os_disk",
		CostComponents: managedDiskCostComponents(region, diskType, diskSizeGB, diskIOPSReadWrite, diskMBPSReadWrite, monthlyDiskOperations),
	}
}

func osDiskSubResource(region, diskType string, diskSizeGB, diskIOPSReadWrite, diskMBPSReadWrite int64, monthlyDiskOperations *decimal.Decimal) *engine.Estimate {
	return &engine.Estimate{
		Name:           "os_disk",
		CostComponents: managedDiskCostComponents(region, diskType, diskSizeGB, diskIOPSReadWrite, diskMBPSReadWrite, monthlyDiskOperations),
	}
}
