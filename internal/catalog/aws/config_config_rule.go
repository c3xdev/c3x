package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/usage"
)

type ConfigConfigRule struct {
	Address                string
	Region                 string
	MonthlyRuleEvaluations *int64 `c3x_usage:"monthly_rule_evaluations"`
}

func (r *ConfigConfigRule) CoreType() string {
	return "ConfigConfigRule"
}

func (r *ConfigConfigRule) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_rule_evaluations", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *ConfigConfigRule) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ConfigConfigRule) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	if r.MonthlyRuleEvaluations != nil {
		monthlyConfigRules := decimal.NewFromInt(*r.MonthlyRuleEvaluations)

		configRulesLimits := []int{100000, 400000}

		rulesTiers := usage.CalculateTierBuckets(monthlyConfigRules, configRulesLimits)

		if rulesTiers[0].GreaterThan(decimal.NewFromInt(0)) {
			costComponents = append(costComponents, r.configRulesCostComponent("Rule evaluations (first 100K)", "0", &rulesTiers[0]))
		}
		if rulesTiers[1].GreaterThan(decimal.NewFromInt(0)) {
			costComponents = append(costComponents, r.configRulesCostComponent("Rule evaluations (next 400K)", "100000", &rulesTiers[1]))
		}
		if rulesTiers[2].GreaterThan(decimal.NewFromInt(0)) {
			costComponents = append(costComponents, r.configRulesCostComponent("Rule evaluations (over 500K)", "500000", &rulesTiers[2]))
		}
	} else {
		var unknown *decimal.Decimal

		costComponents = append(costComponents, r.configRulesCostComponent("Rule evaluations (first 100K)", "0", unknown))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *ConfigConfigRule) configRulesCostComponent(displayName string, usageTier string, monthlyQuantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            displayName,
		Unit:            "evaluations",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyQuantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSConfig"),
			ProductFamily: strPtr("Management Tools - AWS Config Rules"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("^[A-Z0-9]*-ConfigRuleEvaluations$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(usageTier),
		},
		UsageBased: true,
	}
}
