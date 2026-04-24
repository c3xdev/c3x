package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSRecordSetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_dns_record_set",
		CoreRFunc: NewDNSRecordSet,
	}
}

func NewDNSRecordSet(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.DNSRecordSet{
		Address: d.Address,
	}

	return r
}
