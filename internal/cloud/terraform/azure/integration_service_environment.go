package azure

import (
	"strconv"
	"strings"

	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

func GetAzureRMAppIntegrationServiceEnvironmentRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_integration_service_environment",
		RFunc: NewAzureRMIntegrationServiceEnvironment,
	}
}

func NewAzureRMIntegrationServiceEnvironment(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region

	productName := "Logic Apps Integration Service Environment"
	skuName := d.Get("sku_name").String()
	sku := strings.ToLower(skuName[:strings.IndexByte(skuName, '_')])
	scaleNumber, _ := strconv.Atoi(skuName[strings.IndexByte(skuName, '_')+1:])

	costComponents := make([]*engine.LineItem, 0)

	if sku == "developer" {
		productName += " - Developer"
	}

	costComponents = append(costComponents, IntegrationBaseServiceEnvironmentCostComponent("Base units", region, productName))

	if sku == "premium" && scaleNumber > 0 {
		costComponents = append(costComponents, IntegrationScaleServiceEnvironmentCostComponent("Scale units", region, productName, scaleNumber))

	}
	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}

func IntegrationBaseServiceEnvironmentCostComponent(name, region, productName string) *engine.LineItem {
	return &engine.LineItem{

		Name:           name,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Logic Apps"),
			ProductFamily: strPtr("Integration"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr(productName)},
				{Key: "skuName", Value: strPtr("Base")},
				{Key: "meterName", Value: strPtr("Base Unit")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
func IntegrationScaleServiceEnvironmentCostComponent(name, region, productName string, scaleNumber int) *engine.LineItem {
	return &engine.LineItem{

		Name:           name,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(int64(scaleNumber))),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Logic Apps"),
			ProductFamily: strPtr("Integration"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr(productName)},
				{Key: "skuName", Value: strPtr("Scale")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
