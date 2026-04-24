package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"
)

func GetAzureRMNotificationHubNamespaceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_notification_hub_namespace",
		RFunc: NewAzureRMNotificationHubNamespace,
	}
}

func NewAzureRMNotificationHubNamespace(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region

	var monthlyAdditionalPushes *decimal.Decimal
	sku := "Basic"

	if d.Get("sku_name").Type != gjson.Null {
		sku = d.Get("sku_name").String()
	}
	if strings.ToLower(sku) == "free" {
		return &engine.Estimate{
			Name:      d.Address,
			NoPrice:   true,
			IsSkipped: true,
		}
	}

	costComponents := make([]*engine.LineItem, 0)
	costComponents = append(costComponents, notificationHubsCostComponent("Namespace usage", region, sku))
	if u != nil && u.Get("monthly_pushes").Type != gjson.Null {
		monthlyAdditionalPushes = decimalPtr(decimal.NewFromInt(u.Get("monthly_pushes").Int()))
	}

	if strings.ToLower(sku) == "basic" {
		if monthlyAdditionalPushes != nil {
			pushLimits := []int{10000000}
			pushQuantities := usage.CalculateTierBuckets(*monthlyAdditionalPushes, pushLimits)
			if pushQuantities[1].GreaterThan(decimal.Zero) {
				costComponents = append(costComponents, notificationHubsPushesCostComponent("Pushes (over 10M)", region, sku, "10", &pushQuantities[1], 1000000))
			}
		} else {
			costComponents = append(costComponents, notificationHubsPushesCostComponent("Pushes (over 10M)", region, sku, "10", nil, 1000000))
		}
	} else {
		if monthlyAdditionalPushes != nil {
			pushLimits := []int{10000000, 90000000}
			pushQuantities := usage.CalculateTierBuckets(*monthlyAdditionalPushes, pushLimits)
			if pushQuantities[1].GreaterThan(decimal.Zero) {
				costComponents = append(costComponents, notificationHubsPushesCostComponent("Pushes (10-100M)", region, sku, "10", &pushQuantities[1], 1000000))
			}
			if pushQuantities[2].GreaterThan(decimal.Zero) {
				costComponents = append(costComponents, notificationHubsPushesCostComponent("Pushes (over 100M)", region, sku, "100", &pushQuantities[2], 1000000))
			}
		} else {
			costComponents = append(costComponents, notificationHubsPushesCostComponent("Pushes (10-100M)", region, sku, "10", nil, 1000000))
			costComponents = append(costComponents, notificationHubsPushesCostComponent("Pushes (over 100M)", region, sku, "100", nil, 1000000))
		}
	}

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}

func notificationHubsCostComponent(name, region, sku string) *engine.LineItem {
	return &engine.LineItem{
		Name:            fmt.Sprintf("%s (%s)", name, sku),
		Unit:            "months",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("azure"),
			Region:     strPtr(region),
			Service:    strPtr("Notification Hubs"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Notification Hubs")},
				{Key: "skuName", Value: strPtr(sku)},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Unit", sku))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func notificationHubsPushesCostComponent(name, region, sku, startUsageAmt string, quantity *decimal.Decimal, multi int) *engine.LineItem {
	if quantity != nil {
		quantity = decimalPtr(quantity.Div(decimal.NewFromInt(int64(multi))))
	}
	return &engine.LineItem{
		Name:            name,
		Unit:            "1M pushes",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("azure"),
			Region:     strPtr(region),
			Service:    strPtr("Notification Hubs"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Notification Hubs")},
				{Key: "skuName", Value: strPtr(sku)},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Pushes", sku))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr(startUsageAmt),
		},
	}
}
