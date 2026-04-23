package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMLoadBalancerOutboundRuleRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_lb_outbound_rule",
		RFunc: NewAzureRMLoadBalancerOutboundRule,
		ReferenceAttributes: []string{
			"loadbalancer_id",
			"resource_group_name",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"loadbalancer_id", "resource_group_name"})
		},
	}
}

func NewAzureRMLoadBalancerOutboundRule(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region
	region = convertRegion(region)

	lbSku := getParentLbSku(d.References("loadbalancer_id"))

	if lbSku == "" || strings.ToLower(lbSku) == "basic" {
		return &engine.Estimate{
			Name:      d.Address,
			NoPrice:   true,
			IsSkipped: true,
		}
	}

	var costComponents []*engine.LineItem
	costComponents = append(costComponents, rulesCostComponent(region))

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}
