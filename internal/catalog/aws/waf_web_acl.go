package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type WAFWebACL struct {
	Address         string
	Region          string
	Rules           int64
	RuleGroups      int64
	RuleGroupRules  *int64 `c3x_usage:"rule_group_rules"`
	MonthlyRequests *int64 `c3x_usage:"monthly_requests"`
}

func (r *WAFWebACL) CoreType() string {
	return "WAFWebACL"
}

func (r *WAFWebACL) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "rule_group_rules", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_requests", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *WAFWebACL) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *WAFWebACL) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{r.webACLUsageCostComponent()}

	rules := r.Rules
	if r.RuleGroupRules != nil {
		rules += *r.RuleGroupRules
	}

	costComponents = append(costComponents, r.rulesCostComponent(rules))
	costComponents = append(costComponents, r.ruleGroupsCostComponent(r.RuleGroups))
	costComponents = append(costComponents, r.requestsCostComponent())

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *WAFWebACL) webACLUsageCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Web ACL usage",
		Unit:            "months",
		UnitMultiplier:  decimal.NewFromInt(int64(1)),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("awswaf"),
			ProductFamily: strPtr("Web Application Firewall"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/^[A-Z0-9]*-(?!ShieldProtected-)WebACL$/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}
}

func (r *WAFWebACL) rulesCostComponent(rules int64) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Rules",
		Unit:            "rules",
		UnitMultiplier:  decimal.NewFromInt(int64(1)),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(rules)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("awswaf"),
			ProductFamily: strPtr("Web Application Firewall"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/^[A-Z0-9]*-(?!ShieldProtected-)Rule$/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}
}

func (r *WAFWebACL) ruleGroupsCostComponent(ruleGroups int64) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Rule groups",
		Unit:            "groups",
		UnitMultiplier:  decimal.NewFromInt(int64(1)),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(ruleGroups)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("awswaf"),
			ProductFamily: strPtr("Web Application Firewall"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/^[A-Z0-9]*-(?!ShieldProtected-)Rule$/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *WAFWebACL) requestsCostComponent() *engine.LineItem {
	var requests *decimal.Decimal
	if r.MonthlyRequests != nil {
		requests = decimalPtr(decimal.NewFromInt(*r.MonthlyRequests))
	}

	return &engine.LineItem{
		Name:            "Requests",
		Unit:            "1M requests",
		UnitMultiplier:  decimal.NewFromInt(int64(1000000)),
		MonthlyQuantity: requests,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("awswaf"),
			ProductFamily: strPtr("Web Application Firewall"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/^[A-Z0-9]*-(?!ShieldProtected-)Request$/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}
