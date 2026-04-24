package azure

import (
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMAppNATGatewayRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_nat_gateway",
		RFunc: NewAzureRMNATGateway,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func NewAzureRMNATGateway(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region
	region = convertRegion(region)

	var monthlyDataProcessedGb *decimal.Decimal
	if u != nil && u.Get("monthly_data_processed_gb").Type != gjson.Null {
		monthlyDataProcessedGb = decimalPtr(decimal.NewFromFloat(u.Get("monthly_data_processed_gb").Float()))
	}

	costComponents := make([]*engine.LineItem, 0)

	costComponents = append(costComponents, NATGatewayCostComponent("NAT gateway", region))
	costComponents = append(costComponents, DataProcessedCostComponent("Data processed", region, monthlyDataProcessedGb))

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}

func NATGatewayCostComponent(name, region string) *engine.LineItem {
	return &engine.LineItem{

		Name:           name,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("NAT Gateway"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", Value: strPtr("Standard Gateway")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
func DataProcessedCostComponent(name, region string, monthlyDataProcessedGb *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{

		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyDataProcessedGb,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("NAT Gateway"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", Value: strPtr("Standard Data Processed")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
