package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLinuxVirtualMachineRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_linux_virtual_machine",
		CoreRFunc: NewAzureLinuxVirtualMachine,
		Notes: []string{
			"Non-standard images such as RHEL are not supported.",
			"Low priority, Spot and Reserved instances are not supported.",
		},
	}
}
func NewAzureLinuxVirtualMachine(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.LinuxVirtualMachine{
		Address:         d.Address,
		Region:          d.Region,
		Size:            d.Get("size").String(),
		UltraSSDEnabled: d.Get("additional_capabilities.0.ultra_ssd_enabled").Bool(),
	}

	if len(d.Get("os_disk").Array()) > 0 {
		storageData := d.Get("os_disk").Array()[0]
		r.OSDiskData = &azure.ManagedDiskData{
			DiskType:   storageData.Get("storage_account_type").String(),
			DiskSizeGB: storageData.Get("disk_size_gb").Int(),
		}
	}

	return r
}
