package google

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// SecretManagerSecret represents Google Secret Manager's Secret resource.
//
// The cost of active secret versions depends on the number of replication
// locations. If it's more than one then the price is multiplied by the
// locations' quantity. Pricing API includes Free Tier, but it's not used.
//
// More resource information here: https://cloud.google.com/secret-manager
// Pricing information here: https://cloud.google.com/secret-manager/pricing
type SecretManagerSecret struct {
	Address              string
	Region               string
	ReplicationLocations int64

	// "usage" args
	ActiveSecretVersions         *int64 `c3x_usage:"active_secret_versions"`
	MonthlyAccessOperations      *int64 `c3x_usage:"monthly_access_operations"`
	MonthlyRotationNotifications *int64 `c3x_usage:"monthly_rotation_notifications"`
}

func (r *SecretManagerSecret) CoreType() string {
	return "SecretManagerSecret"
}

func (r *SecretManagerSecret) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "active_secret_versions", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_access_operations", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_rotation_notifications", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the SecretManagerSecret.
// It uses the `c3x_usage` struct tags to populate data into the SecretManagerSecret.
func (r *SecretManagerSecret) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid SecretManagerSecret.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *SecretManagerSecret) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	costComponents = append(costComponents, r.activeSecretVersionsCostComponents()...)
	costComponents = append(costComponents, r.accessOperationsCostComponents()...)
	costComponents = append(costComponents, r.rotationNotificationsCostComponents()...)

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

// activeSecretVersionsCostComponents returns a cost component for Active Secret
// Versions.
// The cost is multiplied by the number of replication locations. Free tier
// pricing is excluded.
func (r *SecretManagerSecret) activeSecretVersionsCostComponents() []*engine.LineItem {
	var quantity *int64

	if r.ActiveSecretVersions != nil {
		multiplied := r.ReplicationLocations * *r.ActiveSecretVersions
		quantity = &multiplied
	}

	return []*engine.LineItem{
		{
			Name:            "Active secret versions",
			Unit:            "versions",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: intPtrToDecimalPtr(quantity),
			ProductFilter:   r.buildProductFilter("Secret version replica storage"),
			PriceFilter:     r.buildPriceFilter("6"),
			UsageBased:      true,
		},
	}
}

// accessOperationsCostComponents returns a cost component for Secret's Access
// Operations. Free tier pricing is excluded.
func (r *SecretManagerSecret) accessOperationsCostComponents() []*engine.LineItem {
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

// rotationNotificationsCostComponents returns a cost component for Secret's
// Rotation Notifications. Free tier pricing is excluded.
func (r *SecretManagerSecret) rotationNotificationsCostComponents() []*engine.LineItem {
	return []*engine.LineItem{
		{
			Name:            "Rotation notifications",
			Unit:            "rotations",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: intPtrToDecimalPtr(r.MonthlyRotationNotifications),
			ProductFilter:   r.buildProductFilter("Secret rotate operations"),
			PriceFilter:     r.buildPriceFilter("3"),
			UsageBased:      true,
		},
	}
}

// buildProductFilter creates a product filter for Secret Manager's Secret
// product.
func (r *SecretManagerSecret) buildProductFilter(description string) *engine.ProductSelector {
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
func (r *SecretManagerSecret) buildPriceFilter(startUsageAmount string) *engine.RateSelector {
	return &engine.RateSelector{
		PurchaseOption:   strPtr("OnDemand"),
		StartUsageAmount: strPtr(startUsageAmount),
	}
}
