package azure

import (
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMLoadBalancerRuleRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_lb_rule",
		RFunc: NewAzureRMLoadBalancerRule,
		ReferenceAttributes: []string{
			"loadbalancer_id",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"loadbalancer_id"})
		},
	}
}

func NewAzureRMLoadBalancerRule(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
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

func getParentLbSku(lb []*engine.ResourceSpec) string {
	if len(lb) != 1 {
		return ""
	}

	if lb[0].Get("sku").Type != gjson.Null {
		return lb[0].Get("sku").String()
	}

	return "Basic" // default to basic
}

func rulesCostComponent(region string) *engine.LineItem {
	return &engine.LineItem{
		Name:           "Rule usage",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Load Balancer"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr("Standard")},
				{Key: "meterName", ValueRegex: regexPtr("Overage LB Rules and Outbound Rules$")},
			},
		},
	}
}
