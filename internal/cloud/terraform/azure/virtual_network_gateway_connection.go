package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMVirtualNetworkGatewayConnectionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_virtual_network_gateway_connection",
		RFunc: NewAzureRMVirtualNetworkGatewayConnection,
		ReferenceAttributes: []string{
			"virtual_network_gateway_id",
		},
		Notes: []string{"Price for additional S2S tunnels is used"},
	}
}

func NewAzureRMVirtualNetworkGatewayConnection(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {

	sku := "Basic"

	var vpnGateway *engine.ResourceSpec
	if len(d.References("virtual_network_gateway_id")) > 0 {
		vpnGateway = d.References("virtual_network_gateway_id")[0]
		sku = vpnGateway.Get("sku").String()
	}

	region := d.Region
	if strings.ToLower(sku) == "basic" {
		return &engine.Estimate{
			Name:      d.Address,
			NoPrice:   true,
			IsSkipped: true,
		}
	}
	costComponents := make([]*engine.LineItem, 0)

	if d.Get("type").Type != gjson.Null {
		if strings.ToLower(d.Get("type").String()) == "ipsec" {
			costComponents = append(costComponents, vpnGatewayS2S(region, sku))
		}
	}

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}

func vpnGatewayS2S(region, sku string) *engine.LineItem {
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
				{Key: "skuName", Value: strPtr(sku)},
				{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("/%s/i", "S2S Connection"))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
