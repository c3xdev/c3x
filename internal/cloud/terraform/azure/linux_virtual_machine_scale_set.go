package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLinuxVirtualMachineScaleSetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_linux_virtual_machine_scale_set",
		RFunc: NewLinuxVirtualMachineScaleSet,
	}
}

func NewLinuxVirtualMachineScaleSet(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	r := &azure.LinuxVirtualMachineScaleSet{
		Address:         d.Address,
		Region:          d.Region,
		SKU:             d.Get("sku").String(),
		UltraSSDEnabled: d.Get("additional_capabilities.0.ultra_ssd_enabled").Bool(),
	}

	if len(d.Get("os_disk").Array()) > 0 {
		storageData := d.Get("os_disk").Array()[0]
		r.OSDiskData = &azure.ManagedDiskData{
			DiskType:   storageData.Get("storage_account_type").String(),
			DiskSizeGB: storageData.Get("disk_size_gb").Int(),
		}
	}

	r.PopulateUsage(u)

	if u == nil || u.IsEmpty("instances") {
		r.Instances = intPtr(d.Get("instances").Int())
	}

	return r.BuildResource()
}
