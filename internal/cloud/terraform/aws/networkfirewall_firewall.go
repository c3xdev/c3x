package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNetworkfirewallFirewallRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_networkfirewall_firewall",
		CoreRFunc: newNetworkfirewallFirewall,
	}
}

func newNetworkfirewallFirewall(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	r := &aws.NetworkfirewallFirewall{
		Address: d.Address,
		Region:  region,
	}

	return r
}
