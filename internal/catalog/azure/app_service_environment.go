package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type AppServiceEnvironment struct {
	Address         string
	Region          string
	PricingTier     string
	OperatingSystem *string `c3x_usage:"operating_system"`
}

func (r *AppServiceEnvironment) CoreType() string {
	return "AppServiceEnvironment"
}

func (r *AppServiceEnvironment) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "operating_system", ValueType: engine.String, DefaultValue: "linux"}}
}

func (r *AppServiceEnvironment) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *AppServiceEnvironment) BuildResource() *engine.Estimate {
	tier := "I1"
	if r.PricingTier != "" {
		tier = r.PricingTier
	}

	stampFeeTiers := []string{"I1", "I2", "I3"}
	productName := "Isolated Plan"
	costComponents := make([]*engine.LineItem, 0)
	os := "linux"
	if r.OperatingSystem != nil {
		os = strings.ToLower(*r.OperatingSystem)
	}
	if os == "linux" {
		productName += " - Linux"
	}
	if contains(stampFeeTiers, tier) == bool(true) {
		costComponents = append(costComponents, r.appIsolatedServicePlanCostComponentStampFee(productName))
	}
	costComponents = append(costComponents, r.appIsolatedServicePlanCostComponent(fmt.Sprintf("Instance usage (%s)", tier), productName, tier))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
func (r *AppServiceEnvironment) appIsolatedServicePlanCostComponentStampFee(productName string) *engine.LineItem {
	return &engine.LineItem{

		Name:           "Stamp fee",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure App Service"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Azure App Service " + productName)},
				{Key: "skuName", Value: strPtr("Stamp")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
func (r *AppServiceEnvironment) appIsolatedServicePlanCostComponent(name, productName, tier string) *engine.LineItem {
	return &engine.LineItem{
		Name:           name,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure App Service"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Azure App Service " + productName)},
				{Key: "skuName", Value: strPtr(tier)},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
