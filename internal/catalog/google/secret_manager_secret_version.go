package google

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// SecretManagerSecretVersion represents one Google Secret Manager Secret's Version resource.
//
// The cost of active secret version depends on the number of replication
// locations specified by its parent secret. If it's more than one then the price
// is multiplied by the locations' quantity.
// Pricing API includes Free Tier, but it's not used.
//
// More resource information here: https://cloud.google.com/secret-manager
// Pricing information here: https://cloud.google.com/secret-manager/pricing
type SecretManagerSecretVersion struct {
	Address              string
	Region               string
	ReplicationLocations int64

	// "usage" args
	MonthlyAccessOperations *int64 `c3x_usage:"monthly_access_operations"`
}

func (r *SecretManagerSecretVersion) CoreType() string {
	return "SecretManagerSecretVersion"
}

func (r *SecretManagerSecretVersion) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_access_operations", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the SecretManagerSecretVersion.
// It uses the `c3x_usage` struct tags to populate data into the SecretManagerSecretVersion.
func (r *SecretManagerSecretVersion) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid SecretManagerSecretVersion.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *SecretManagerSecretVersion) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	costComponents = append(costComponents, r.activeSecretVersionsCostComponents()...)
	costComponents = append(costComponents, r.accessOperationsCostComponents()...)

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

// activeSecretVersionsCostComponents returns a cost component for the Active Secret
// Version. By default it represents one version.
// The cost is multiplied by the number of replication locations. Free tier
// pricing is excluded.
func (r *SecretManagerSecretVersion) activeSecretVersionsCostComponents() []*engine.LineItem {
	return []*engine.LineItem{
		{
			Name:            "Active secret versions",
			Unit:            "versions",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: intPtrToDecimalPtr(&r.ReplicationLocations),
			ProductFilter:   r.buildProductFilter("Secret version replica storage"),
			PriceFilter:     r.buildPriceFilter("6"),
		},
	}
}

// accessOperationsCostComponents returns a cost component for Secret Version's Access
// Operations. Free tier pricing is excluded.
func (r *SecretManagerSecretVersion) accessOperationsCostComponents() []*engine.LineItem {
	multiplier := 10000

	return []*engine.LineItem{
		{
			Name:            "Access operations",
			Unit:            "10K requests",
			UnitMultiplier:  decimal.NewFromInt(int64(multiplier)),
			MonthlyQuantity: intPtrToDecimalPtr(r.MonthlyAccessOperations),
			ProductFilter:   r.buildProductFilter("Secret access operations"),
			PriceFilter:     r.buildPriceFilter(fmt.Sprint(multiplier)),
			UsageBased:      true,
		},
	}
}

// buildProductFilter creates a product filter for Secret Manager's Secret
// product.
func (r *SecretManagerSecretVersion) buildProductFilter(description string) *engine.ProductSelector {
	return &engine.ProductSelector{
		VendorName:    strPtr("gcp"),
		Region:        strPtr(r.Region),
		Service:       strPtr("Secret Manager"),
		ProductFamily: strPtr("ApplicationServices"),
		AttributeFilters: []*engine.AttributeMatch{
			{Key: "description", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", description))},
		},
	}
}

// buildPriceFilter creates a price filter based on start usage amount to ignore
// free tier pricing.
func (r *SecretManagerSecretVersion) buildPriceFilter(startUsageAmount string) *engine.RateSelector {
	return &engine.RateSelector{
		PurchaseOption:   strPtr("OnDemand"),
		StartUsageAmount: strPtr(startUsageAmount),
	}
}
