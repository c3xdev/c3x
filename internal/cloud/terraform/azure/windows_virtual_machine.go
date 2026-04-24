package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getWindowsVirtualMachineRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_windows_virtual_machine",
		CoreRFunc: NewWindowsVirtualMachine,
		Notes: []string{
			"Low priority, Spot and Reserved instances are not supported.",
		},
	}
}

func NewWindowsVirtualMachine(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.WindowsVirtualMachine{
		Address:                               d.Address,
		Region:                                d.Region,
		Size:                                  d.Get("size").String(),
		LicenseType:                           d.Get("license_type").String(),
		AdditionalCapabilitiesUltraSSDEnabled: d.Get("additional_capabilities.0.ultra_ssd_enabled").Bool(),
		IsDevTest:                             d.ProjectMetadata["isProduction"] == "false",
	}
	if len(d.Get("os_disk").Array()) > 0 {
		diskData := d.Get("os_disk").Array()[0]
		r.OSDiskData = &azure.ManagedDiskData{
			DiskType:   diskData.Get("storage_account_type").String(),
			DiskSizeGB: diskData.Get("disk_size_gb").Int(),
		}
	}
	return r
}
