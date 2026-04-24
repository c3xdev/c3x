package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

// getFrontdoorFirewallPolicyRegistryItem returns a registry item for the
// resource
func getFrontdoorFirewallPolicyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_frontdoor_firewall_policy",
		CoreRFunc: newFrontdoorFirewallPolicy,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

// newFrontdoorFirewallPolicy parses Terraform's data and uses it to build
// a new resource
func newFrontdoorFirewallPolicy(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	if strings.HasPrefix(strings.ToLower(region), "usgov") {
		region = "US Gov Zone 1"
	} else {
		region = regionToCDNZone(region)
	}

	customRules := 0
	if rules := d.Get("custom_rule"); rules.Exists() {
		customRules = len(rules.Array())
	}

	managedRulesets := 0
	if rules := d.Get("managed_rule"); rules.Exists() {
		managedRulesets = len(rules.Array())
	}

	r := &azure.FrontdoorFirewallPolicy{
		Address:         d.Address,
		Region:          region,
		CustomRules:     customRules,
		ManagedRulesets: managedRulesets,
	}
	return r
}
