package azure

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"
)

func GetAzureRMVirtualNetworkGatewayRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_virtual_network_gateway",
		RFunc: NewAzureRMVirtualNetworkGateway,
		Notes: []string{},
	}
}

func NewAzureRMVirtualNetworkGateway(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	var connection, dataTransfers *decimal.Decimal
	sku := "Basic"
	region := d.Region
	zone := regionToVNETZone(region)

	if d.Get("sku").Type != gjson.Null {
		sku = d.Get("sku").String()
	}
	meterName := sku

	costComponents := make([]*engine.LineItem, 0)

	if sku == "Basic" {
		meterName = "Basic Gateway"
	}

	costComponents = append(costComponents, vpnGateway(region, sku, meterName))

	if u != nil && u.Get("p2s_connection").Type != gjson.Null {
		connection = decimalPtr(decimal.NewFromInt(u.Get("p2s_connection").Int()))
		if connection != nil {
			connectionLimits := []int{128}
			connectionValues := usage.CalculateTierBuckets(*connection, connectionLimits)
			if connectionValues[1].GreaterThan(decimal.Zero) {
				costComponents = append(costComponents, vpnGatewayP2S(region, sku, &connectionValues[1]))
			}
		}
	} else {
		costComponents = append(costComponents, vpnGatewayP2S(region, sku, connection))
	}

	if u != nil && u.Get("monthly_data_transfer_gb").Type != gjson.Null {
		dataTransfers = decimalPtr(decimal.NewFromInt(u.Get("monthly_data_transfer_gb").Int()))
		if dataTransfers != nil {
			costComponents = append(costComponents, vpnGatewayDataTransfers(zone, sku, dataTransfers))
		}
	} else {
		costComponents = append(costComponents, vpnGatewayDataTransfers(zone, sku, dataTransfers))
	}

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}

func vpnGateway(region, sku, meterName string) *engine.LineItem {
	return &engine.LineItem{
		Name:           fmt.Sprintf("VPN gateway (%s)", sku),
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("azure"),
			Region:     strPtr(region),
			Service:    strPtr("VPN Gateway"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", Value: strPtr(meterName)},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func vpnGatewayP2S(region, sku string, connection *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:           "VPN gateway P2S tunnels (over 128)",
		Unit:           "tunnel",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: connection,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("azure"),
			Region:     strPtr(region),
			Service:    strPtr("VPN Gateway"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(sku)},
				{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("/%s/i", "P2S Connection"))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func vpnGatewayDataTransfers(zone, sku string, dataTransfers *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "VPN gateway data tranfer",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: dataTransfers,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("azure"),
			Region:     strPtr(zone),
			Service:    strPtr("VPN Gateway"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "serviceFamily", ValueRegex: strPtr(fmt.Sprintf("/%s/i", "Networking"))},
				{Key: "productName", ValueRegex: strPtr(fmt.Sprintf("/%s/i", "VPN Gateway Bandwidth"))},
				{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("/%s/i", "Inter-Virtual Network Data Transfer Out"))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
