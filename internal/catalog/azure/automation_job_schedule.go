package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type AutomationJobSchedule struct {
	Address           string
	Region            string
	MonthlyJobRunMins *int64 `c3x_usage:"monthly_job_run_mins"`
}

func (r *AutomationJobSchedule) CoreType() string {
	return "AutomationJobSchedule"
}

func (r *AutomationJobSchedule) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_job_run_mins", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *AutomationJobSchedule) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *AutomationJobSchedule) BuildResource() *engine.Estimate {
	var monthlyJobRunMins *decimal.Decimal
	location := r.Region

	if r.MonthlyJobRunMins != nil {
		monthlyJobRunMins = decimalPtr(decimal.NewFromInt(*r.MonthlyJobRunMins))
	}

	costComponents := make([]*engine.LineItem, 0)
	costComponents = append(costComponents, automationRunTimeCostComponent(location, "500", "Basic Runtime", "Basic", monthlyJobRunMins))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func automationRunTimeCostComponent(location, startUsage, meterName, skuName string, monthlyQuantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{

		Name:            "Job run time",
		Unit:            "minutes",
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
