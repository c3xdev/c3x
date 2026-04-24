package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getWindowsVirtualMachineScaleSetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_windows_virtual_machine_scale_set",
		RFunc: NewWindowsVirtualMachineScaleSet,
	}
}
func NewWindowsVirtualMachineScaleSet(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	r := &azure.WindowsVirtualMachineScaleSet{
		Address:                               d.Address,
		Region:                                d.Region,
		SKU:                                   d.Get("sku").String(),
		LicenseType:                           d.Get("license_type").String(),
		IsDevTest:                             d.ProjectMetadata["isProduction"] == "false",
		AdditionalCapabilitiesUltraSSDEnabled: d.Get("additional_capabilities.0.ultra_ssd_enabled").Bool(),
	}
	if len(d.Get("os_disk").Array()) > 0 {
		diskData := d.Get("os_disk").Array()[0]
		r.OSDiskData = &azure.ManagedDiskData{
			DiskType:   diskData.Get("storage_account_type").String(),
			DiskSizeGB: diskData.Get("disk_size_gb").Int(),
		}
	}

	r.PopulateUsage(u)

	if u == nil || u.IsEmpty("instances") {
		r.Instances = intPtr(d.Get("instances").Int())
	}

	return r.BuildResource()
}
