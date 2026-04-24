package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSnapshotRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_snapshot",
		CoreRFunc: newSnapshot,
		ReferenceAttributes: []string{
			"resource_group_name",
			"source_uri",
		},
	}
}

func newSnapshot(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	return &azure.Image{
		Type:      "Snapshot",
		StorageGB: snapshotStorageSize(d),
		Address:   d.Address,
		Region:    region,
	}
}

func snapshotStorageSize(d *engine.ResourceSpec) *float64 {
	v := d.Get("disk_size_gb")
	if v.Exists() && v.Value() != nil {
		size := v.Float()
		return &size
	}

	refs := d.References("source_uri")
	if len(refs) > 0 {
		size := refs[0].Get("disk_size_gb").Float()
		return &size
	}

	return nil
}
