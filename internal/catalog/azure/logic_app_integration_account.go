package azure

import (
	"fmt"

	"github.com/shopspring/decimal"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// LogicAppIntegrationAccount struct represents Microsoft's cloud-based solution for
// integrating business functions and data sources.
//
// Resource information:https://learn.microsoft.com/en-us/azure/logic-apps/logic-apps-pricing
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/logic-apps/
type LogicAppIntegrationAccount struct {
	Address string
	Region  string
	SKU     string
}

// NewLogicAppIntegrationAccount returns an initialised LogicAppIntegrationAccount with the provided attributes.
// This should be used over simple struct initialisation as NewLogicAppIntegrationAccount ensures that the casing
// for the SKU is consistent.
func NewLogicAppIntegrationAccount(address string, region string, sku string) *LogicAppIntegrationAccount {
	return &LogicAppIntegrationAccount{
		Address: address,
		Region:  region,
		SKU:     cases.Title(language.English).String(sku),
	}
}

// CoreType returns the name of this resource type
func (r *LogicAppIntegrationAccount) CoreType() string {
	return "LogicAppIntegrationAccount"
}

// UsageSchema defines a list which represents the usage schema of LogicAppIntegrationAccount.
func (r *LogicAppIntegrationAccount) UsageSchema() []*engine.ConsumptionField {
	return nil
}

// PopulateUsage parses the u engine.ConsumptionProfile into the LogicAppIntegrationAccount.
// It uses the `c3x_usage` struct tags to populate data into the LogicAppIntegrationAccount.
func (r *LogicAppIntegrationAccount) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid LogicAppIntegrationAccount struct.
//
// LogicAppIntegrationAccount only have one associated cost with them which is the hourly cost of the account.
// The integration is billed hourly but the prices available are monthly. Therefore, we use the MonthToHourUnitMultiplier
// to convert this price to a more "correct" unit.
func (r *LogicAppIntegrationAccount) BuildResource() *engine.Estimate {
	if r.SKU == "Free" {
		return &engine.Estimate{
			Name:        r.Address,
			UsageSchema: r.UsageSchema(),
			NoPrice:     true,
		}
	}

	var rounding int32 = 0

	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
		CostComponents: []*engine.LineItem{
			{
				Name:            fmt.Sprintf("Integration Account (%s)", r.SKU),
				Unit:            "hours",
				MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				UnitMultiplier:  engine.MonthToHourUnitMultiplier,
				UnitRounding:    &rounding,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("azure"),
					Region:        strPtr(r.Region),
					Service:       strPtr("Logic Apps"),
					ProductFamily: strPtr("Integration"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Unit", r.SKU))},
						{Key: "skuName", Value: strPtr(r.SKU)},
						{Key: "productName", Value: strPtr("Logic Apps Integration Account")},
					},
				},
				PriceFilter: &engine.RateSelector{
					PurchaseOption: strPtr("Consumption"),
				},
			},
		},
	}
}
