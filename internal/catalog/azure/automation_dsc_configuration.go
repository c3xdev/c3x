package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type AutomationDSCConfiguration struct {
	Address                 string
	Region                  string
	NonAzureConfigNodeCount *int64 `c3x_usage:"non_azure_config_node_count"`
}

func (r *AutomationDSCConfiguration) CoreType() string {
	return "AutomationDSCConfiguration"
}

func (r *AutomationDSCConfiguration) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "non_azure_config_node_count", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *AutomationDSCConfiguration) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *AutomationDSCConfiguration) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: automationDSCNodesCostComponent(&r.Region, r.NonAzureConfigNodeCount),
		UsageSchema:    r.UsageSchema(),
	}
}

func automationDSCNodesCostComponent(location *string, nonAzureConfigNodeCount *int64) []*engine.LineItem {
	var nonAzureConfigNodeCountDec *decimal.Decimal

	if nonAzureConfigNodeCount != nil {
		nonAzureConfigNodeCountDec = decimalPtr(decimal.NewFromInt(*nonAzureConfigNodeCount))
	}

	costComponents := make([]*engine.LineItem, 0)
	costComponents = append(costComponents, nonautomationDSCNodesCostComponent(*location, "5", "Non-Azure Node", "Non-Azure", nonAzureConfigNodeCountDec))

	return costComponents
}

func nonautomationDSCNodesCostComponent(location, startUsage, meterName, skuName string, monthlyQuantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{

		Name:            "Non-azure config nodes",
		Unit:            "nodes",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyQuantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(location),
			Service:       strPtr("Automation"),
			ProductFamily: strPtr("Management and Governance"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", meterName))},
				{Key: "skuName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", skuName))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr(startUsage),
		},
		UsageBased: true,
	}
}
