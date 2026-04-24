package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getIoTHubRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_iothub",
		CoreRFunc: newIoTHub,
	}
}

func getIoTHubDPSRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_iothub_dps",
		CoreRFunc: newIoTHubDPS,
	}
}

func newIoTHub(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	sku := d.Get("sku.0.name").String()
	capacity := d.Get("sku.0.capacity").Int()

	r := &azure.IoTHub{
		Address:  d.Address,
		Region:   region,
		Sku:      sku,
		Capacity: capacity,
	}

	return r
}

func newIoTHubDPS(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	sku := d.Get("sku.0.name").String()

	r := &azure.IoTHubDPS{
		Address: d.Address,
		Region:  region,
		Sku:     sku,
	}

	return r
}
