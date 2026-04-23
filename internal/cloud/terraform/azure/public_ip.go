package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMPublicIPRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_public_ip",
		RFunc: NewAzureRMPublicIP,
	}
}

func NewAzureRMPublicIP(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region

	var meterName string
	sku := "Standard" // default sku is Standard
	skuTier := "Regional"

	if d.Get("sku").Type != gjson.Null {
		sku = d.Get("sku").String()
	}

	allocationMethod := strings.ToLower(d.Get("allocation_method").String())
	if allocationMethod == "dynamic" {
		// dynamic IPs imply a basic sku.
		sku = "Basic"
	}

	switch sku {
	case "Basic":
		meterName = "Basic IPv4 " + allocationMethod + " Public IP"
	case "Standard":
		skuTierVal := d.Get("sku_tier").String()
		if skuTierVal != "" {
			skuTier = skuTierVal
		}
		if skuTier == "Global" {
			sku = "Global" // When sku_tier is Global, skuname is global
			meterName = "Global IPv4 " + allocationMethod + " Public IP"
		} else {
			meterName = "Standard IPv4 " + allocationMethod + " Public IP"
		}
	}

	name := fmt.Sprintf("IP address (%s, %s)", strings.ToLower(allocationMethod), strings.ToLower(skuTier))

	costComponents := make([]*engine.LineItem, 0)

	costComponents = append(costComponents, PublicIPCostComponent(name, region, sku, meterName))

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}
func PublicIPCostComponent(name, region, sku, meterName string) *engine.LineItem {
	return &engine.LineItem{
		Name:           name,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Virtual Network"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("IP Addresses")},
				{Key: "skuName", Value: strPtr(sku)},
				{Key: "meterName", ValueRegex: regexPtr(meterName)},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
