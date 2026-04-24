package azure

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMKeyVaultManagedHSMRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_key_vault_managed_hardware_security_module",
		RFunc: NewAzureRMKeyVaultManagedHSM,
	}
}

func NewAzureRMKeyVaultManagedHSM(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region

	var costComponents []*engine.LineItem

	costComponents = append(costComponents, &engine.LineItem{
		Name:           "HSM pools",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Key Vault"),
			ProductFamily: strPtr("Security"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Azure Dedicated HSM")},
				{Key: "skuName", Value: strPtr("Standard")},
				{Key: "meterName", Value: strPtr("Standard Instance")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
	})

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}
