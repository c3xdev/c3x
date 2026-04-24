package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMEventHubsNamespaceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_eventhub_namespace",
		RFunc: NewAzureRMEventHubs,
		Notes: []string{"Premium namespaces are not supported by Terraform."},
	}
}

func NewAzureRMEventHubs(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	var events, unknown *decimal.Decimal
	var includedRetention decimal.Decimal
	sku := "Basic"
	meterName := ""
	region := d.Region

	if d.Get("sku").Type != gjson.Null {
		sku = d.Get("sku").String()
	}

	capacity := decimal.NewFromInt(1)
	if u != nil && u.Get("throughput_or_capacity_units").Type != gjson.Null {
		capacity = decimal.NewFromInt(u.Get("throughput_or_capacity_units").Int())
	} else if d.Get("capacity").Type != gjson.Null {
		capacity = decimal.NewFromInt(d.Get("capacity").Int())
	}

	if d.Get("dedicated_cluster_id").Type != gjson.Null && len(d.Get("dedicated_cluster_id").String()) > 0 {
		sku = "Dedicated"
		meterName = "Dedicated Capacity Unit"
		includedRetention = capacity.Mul(decimal.NewFromInt(10000))
	}

	if strings.ToLower(sku) == "premium" {
		meterName = "Processing Unit"
		includedRetention = capacity.Mul(decimal.NewFromInt(1000))
	}

	costComponents := make([]*engine.LineItem, 0)

	if u != nil && u.Get("monthly_ingress_events").Type != gjson.Null {
		events = decimalPtr(decimal.NewFromInt(u.Get("monthly_ingress_events").Int()))
	}

	if strings.ToLower(sku) == "basic" {
		meterName = "Basic Throughput Unit"
		costComponents = append(costComponents, eventHubsIngressCostComponent(region, sku, events))
	}

	if strings.ToLower(sku) == "standard" {
		meterName = "Standard Throughput Unit"
		costComponents = append(costComponents, eventHubsIngressCostComponent(region, sku, events))
		if u != nil && u.Get("capture_enabled").Type != gjson.Null && strings.ToLower(u.Get("capture_enabled").String()) == "true" {
			costComponents = append(costComponents, eventHubsCaptureCostComponent(region, sku, capacity))
		}
	}

	costComponents = append(costComponents, eventHubsThroughPutCostComponent(region, sku, meterName, capacity))
	if strings.ToLower(sku) == "dedicated" || strings.ToLower(sku) == "premium" {
		if u != nil && u.Get("retention_storage_gb").Type != gjson.Null {
			retention := decimalPtr(decimal.NewFromInt(u.Get("retention_storage_gb").Int()))
			// Subtract the 10 TB per capacity unit that's included in the dedicated namespace tier
			extendedRetention := retention.Sub(includedRetention)
			if extendedRetention.GreaterThan(decimal.NewFromInt(0)) {
				costComponents = append(costComponents, eventHubsExtensionRetentionCostComponent(region, sku, &extendedRetention))
			}
		} else {
			costComponents = append(costComponents, eventHubsExtensionRetentionCostComponent(region, sku, unknown))
		}
	}

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}

func eventHubsIngressCostComponent(region, sku string, quantity *decimal.Decimal) *engine.LineItem {
	if quantity != nil {
		quantity = decimalPtr(quantity.Div(decimal.NewFromInt(int64(1000000))))
	}
	return &engine.LineItem{
		Name:            fmt.Sprintf("Ingress event (%s)", sku),
		Unit:            "1M events",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Event Hubs"),
			ProductFamily: strPtr("Internet of Things"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", sku))},
				{Key: "meterName", ValueRegex: regexPtr("Ingress Events$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func eventHubsThroughPutCostComponent(region, sku, meterName string, capacity decimal.Decimal) *engine.LineItem {
	unitName := strings.TrimPrefix(strings.ToLower(meterName), strings.ToLower(sku+" ")) + "s"

	return &engine.LineItem{
		Name:           fmt.Sprintf("Capacity (%s)", sku),
		Unit:           unitName,
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: decimalPtr(capacity),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Event Hubs"),
			ProductFamily: strPtr("Internet of Things"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", sku))},
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("%s$", meterName))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func eventHubsCaptureCostComponent(region, sku string, quantity decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:           "Capture",
		Unit:           "units",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: decimalPtr(quantity),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("azure"),
			Region:     strPtr(region),
			Service:    strPtr("Event Hubs"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", sku))},
				{Key: "meterName", ValueRegex: regexPtr("Capture$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func eventHubsExtensionRetentionCostComponent(region, sku string, retention *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Extended retention",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: retention,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("azure"),
			Region:     strPtr(region),
			Service:    strPtr("Event Hubs"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", sku))},
				{Key: "meterName", ValueRegex: regexPtr("Extended Retention$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
