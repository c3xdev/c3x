package azure

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// FrontdoorFirewallPolicy represents a policy for Web Application Firewall (WAF)
// with Azure Front Door.
//
// More resource information here: https://docs.microsoft.com/en-us/azure/web-application-firewall/afds/waf-front-door-drs
// Pricing information here: https://azure.microsoft.com/en-us/pricing/details/frontdoor/#overview
type FrontdoorFirewallPolicy struct {
	Address string
	Region  string

	CustomRules     int
	ManagedRulesets int

	// "usage" args
	MonthlyCustomRuleRequests     *int64 `c3x_usage:"monthly_custom_rule_requests"`
	MonthlyManagedRulesetRequests *int64 `c3x_usage:"monthly_managed_ruleset_requests"`
}

// CoreType returns the name of this resource type
func (r *FrontdoorFirewallPolicy) CoreType() string {
	return "FrontdoorFirewallPolicy"
}

// UsageSchema defines a list which represents the usage schema of EventGridTopic.
func (r *FrontdoorFirewallPolicy) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_custom_rule_requests", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_managed_ruleset_requests", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the FrontdoorFirewallPolicy.
func (r *FrontdoorFirewallPolicy) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid FrontdoorFirewallPolicy.
// This method is called after the resource is initialised by an IaC provider.
func (r *FrontdoorFirewallPolicy) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	costComponents = append(costComponents, r.policyCostComponents()...)
	costComponents = append(costComponents, r.customRulesCostComponents()...)
	costComponents = append(costComponents, r.customRuleRequestsCostComponents()...)
	costComponents = append(costComponents, r.managedRulesetsCostComponents()...)
	costComponents = append(costComponents, r.managedRulesetRequestsCostComponents()...)

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

// policyCostComponents returns cost components for Policy usage
func (r *FrontdoorFirewallPolicy) policyCostComponents() []*engine.LineItem {
	return []*engine.LineItem{
		{
			Name:            "Policy",
			Unit:            "months",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter:   r.buildProductFilter("Policy"),
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		},
	}
}

// customRulesCostComponents returns a cost component for the total number of custom
// rules in the policy.
func (r *FrontdoorFirewallPolicy) customRulesCostComponents() []*engine.LineItem {
	return []*engine.LineItem{
		{
			Name:            "Custom rules",
			Unit:            "rules",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: decimalPtr(decimal.NewFromInt(int64(r.CustomRules))),
			ProductFilter:   r.buildProductFilter("Rule"),
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		},
	}
}

// customRuleRequestsCostComponents returns a usage based cost component for the
// number of custom rules' requests.
func (r *FrontdoorFirewallPolicy) customRuleRequestsCostComponents() []*engine.LineItem {
	return []*engine.LineItem{
		{
			Name:            "Custom rule requests",
			Unit:            "1M requests",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: r.monthlyRequestsQuantity(r.MonthlyCustomRuleRequests),
			ProductFilter:   r.buildProductFilter("Requests"),
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
			UsageBased: true,
		},
	}
}

// managedRulesetsCostComponents returns a cost component for the total number
// of managed rulesets in the policy.
func (r *FrontdoorFirewallPolicy) managedRulesetsCostComponents() []*engine.LineItem {
	return []*engine.LineItem{
		{
			Name:            "Managed rulesets",
			Unit:            "rulesets",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: decimalPtr(decimal.NewFromInt(int64(r.ManagedRulesets))),
			ProductFilter:   r.buildProductFilter("Default Ruleset"),
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		},
	}
}

// managedRulesetRequestsCostComponents returns a usage based cost component for
// the number of managed rulesets' requests.
func (r *FrontdoorFirewallPolicy) managedRulesetRequestsCostComponents() []*engine.LineItem {
	return []*engine.LineItem{
		{
			Name:            "Managed ruleset requests",
			Unit:            "1M requests",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: r.monthlyRequestsQuantity(r.MonthlyManagedRulesetRequests),
			ProductFilter:   r.buildProductFilter("Default Request"),
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
			UsageBased: true,
		},
	}
}

// buildProductFilter returns a product filter for the Front Door's products.
//
// skuName and productName define the original Front Door service (not
// Standard/Premium).
func (r *FrontdoorFirewallPolicy) buildProductFilter(meterName string) *engine.ProductSelector {
	return &engine.ProductSelector{
		VendorName:    strPtr("azure"),
		Region:        strPtr(r.Region),
		Service:       strPtr("Azure Front Door Service"),
		ProductFamily: strPtr("Networking"),
		AttributeFilters: []*engine.AttributeMatch{
			{Key: "skuName", Value: strPtr("Standard")},
			{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("%s$", meterName))},
			{Key: "productName", Value: strPtr("Azure Front Door Service")},
		},
	}
}

// monthlyRequestsQuantity converts the monthly requests usage number as
// Azure's requests pricing is 1M requests/month.
func (r *FrontdoorFirewallPolicy) monthlyRequestsQuantity(requestsNumber *int64) *decimal.Decimal {
	var monthlyRequests *decimal.Decimal
	divider := decimal.NewFromInt(1000000)

	if requestsNumber != nil {
		requests := decimal.NewFromInt(*requestsNumber)
		monthlyRequests = decimalPtr(requests.Div(divider))
	}

	return monthlyRequests
}
