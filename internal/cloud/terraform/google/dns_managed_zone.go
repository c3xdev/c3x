package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSManagedZoneRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_dns_managed_zone",
		CoreRFunc: NewDNSManagedZone,
	}
}

func NewDNSManagedZone(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.DNSManagedZone{
		Address: d.Address,
	}

	return r
}
