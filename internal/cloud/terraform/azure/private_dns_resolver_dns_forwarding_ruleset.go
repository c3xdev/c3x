package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPrivateDnsResolverDnsForwardingRulesetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_resolver_dns_forwarding_ruleset",
		CoreRFunc: newPrivateDnsResolverDnsForwardingRuleset,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newPrivateDnsResolverDnsForwardingRuleset(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	if strings.HasPrefix(strings.ToLower(region), "usgov") {
		region = "US Gov Zone 1"
	} else if strings.HasPrefix(strings.ToLower(region), "germany") {
		region = "DE Zone 1"
	} else if strings.HasPrefix(strings.ToLower(region), "china") {
		region = "Zone 1 (China)"
	} else {
		region = "Zone 1"
	}

	return &azure.PrivateDnsResolverDnsForwardingRuleset{
		Address: d.Address,
		Region:  region,
	}
}
