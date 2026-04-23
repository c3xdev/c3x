package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type AutomationAccount struct {
	Address                 string
	Region                  string
	MonthlyJobRunMins       *int64 `c3x_usage:"monthly_job_run_mins"`
	NonAzureConfigNodeCount *int64 `c3x_usage:"non_azure_config_node_count"`
	MonthlyWatcherHrs       *int64 `c3x_usage:"monthly_watcher_hrs"`
}

func (r *AutomationAccount) CoreType() string {
	return "AutomationAccount"
}

func (r *AutomationAccount) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_job_run_mins", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "non_azure_config_node_count", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_watcher_hrs", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *AutomationAccount) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *AutomationAccount) BuildResource() *engine.Estimate {
	var monthlyJobRunMins, nonAzureConfigNodeCount *decimal.Decimal
	location := r.Region
	costComponents := make([]*engine.LineItem, 0)

	if r.MonthlyJobRunMins != nil {
		monthlyJobRunMins = decimalPtr(decimal.NewFromInt(*r.MonthlyJobRunMins))
		if monthlyJobRunMins.IsPositive() {
			costComponents = append(costComponents, automationRunTimeCostComponent(location, "500", "Basic Runtime", "Basic", monthlyJobRunMins))
		}
	} else {
		costComponents = append(costComponents, automationRunTimeCostComponent(location, "500", "Basic Runtime", "Basic", monthlyJobRunMins))
	}

	if r.NonAzureConfigNodeCount != nil {
		nonAzureConfigNodeCount = decimalPtr(decimal.NewFromInt(*r.NonAzureConfigNodeCount))
		if nonAzureConfigNodeCount.IsPositive() {
			costComponents = append(costComponents, nonautomationDSCNodesCostComponent(location, "5", "Non-Azure Node", "Non-Azure", nonAzureConfigNodeCount))
		}
	} else {
		costComponents = append(costComponents, nonautomationDSCNodesCostComponent(location, "5", "Non-Azure Node", "Non-Azure", nonAzureConfigNodeCount))
	}

	costComponents = append(costComponents, r.watchersCostComponent("744", "Watcher", "Basic"))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *AutomationAccount) watchersCostComponent(startUsage, meterName, skuName string) *engine.LineItem {
	var monthlyQuantity *decimal.Decimal
	if r.MonthlyWatcherHrs != nil {
		monthlyQuantity = decimalPtr(decimal.NewFromInt(*r.MonthlyWatcherHrs))
	}

	return &engine.LineItem{

		Name:            "Watchers",
		Unit:            "hours",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyQuantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
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
