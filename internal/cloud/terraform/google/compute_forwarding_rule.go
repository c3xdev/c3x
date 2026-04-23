package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeForwardingRuleRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_forwarding_rule",
		CoreRFunc: NewComputeForwardingRule,
		Notes:     []string{"Price for additional forwarding rule is used"},
	}
}
func getComputeGlobalForwardingRuleRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_global_forwarding_rule",
		CoreRFunc: NewComputeForwardingRule,
		Notes:     []string{"Price for additional forwarding rule is used"},
	}
}

func NewComputeForwardingRule(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.ComputeForwardingRule{Address: d.Address, Region: d.Get("region").String()}
	return r
}
