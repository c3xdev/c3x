package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getManagedDiskRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_managed_disk",
		CoreRFunc: NewManagedDisk,
	}
}

func NewManagedDisk(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.ManagedDisk{
		Address: d.Address,
		Region:  d.Region,
		ManagedDiskData: azure.ManagedDiskData{
			DiskType:          d.Get("storage_account_type").String(),
			DiskSizeGB:        d.Get("disk_size_gb").Int(),
			DiskIOPSReadWrite: d.Get("disk_iops_read_write").Int(),
			DiskMBPSReadWrite: d.Get("disk_mbps_read_write").Int(),
		},
	}

	return r
}
