package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeAddressRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_address",
		CoreRFunc: newComputeAddress,
		ReferenceAttributes: []string{
			"google_compute_instance.network_interface.0.access_config.0.nat_ip",
		},
	}
}
func getComputeGlobalAddressRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_global_address",
		CoreRFunc: newComputeAddress,
		ReferenceAttributes: []string{
			"google_compute_instance.network_interface.0.access_config.0.nat_ip",
		},
	}
}

func newComputeAddress(d *engine.ResourceSpec) engine.CatalogItem {
	purchaseOption := ""
	instanceRefs := d.References("google_compute_instance.network_interface.0.access_config.0.nat_ip")
	if len(instanceRefs) > 0 {
		purchaseOption = getComputePurchaseOption(instanceRefs[0].RawValues)
	}

	r := &google.ComputeAddress{
		Address:                d.Address,
		Region:                 d.Get("region").String(),
		AddressType:            d.Get("address_type").String(),
		Purpose:                d.Get("purpose").String(),
		InstancePurchaseOption: purchaseOption,
	}
	return r
}
