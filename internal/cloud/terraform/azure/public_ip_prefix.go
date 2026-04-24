package azure

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMPublicIPPrefixRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_public_ip_prefix",
		RFunc: NewAzureRMPublicIPPrefix,
	}
}

func NewAzureRMPublicIPPrefix(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region

	costComponents := make([]*engine.LineItem, 0)

	costComponents = append(costComponents, PublicIPPrefixCostComponent("IP prefix", region))

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}
func PublicIPPrefixCostComponent(name, region string) *engine.LineItem {
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
				{Key: "productName", Value: strPtr("Public IP Prefix")},
				{Key: "meterName", ValueRegex: strPtr("/Static IP Addresses/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
