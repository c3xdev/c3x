package aws

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type NeptuneClusterInstance struct {
	Address             string
	Region              string
	InstanceClass       string
	IOOptimized         bool
	Count               *int64
	MonthlyCPUCreditHrs *int64 `c3x_usage:"monthly_cpu_credit_hrs"`
}

func (r *NeptuneClusterInstance) CoreType() string {
	return "NeptuneClusterInstance"
}

func (r *NeptuneClusterInstance) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_cpu_credit_hrs", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *NeptuneClusterInstance) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *NeptuneClusterInstance) BuildResource() *engine.Estimate {
	hourlyQuantity := 1
	if r.Count != nil {
		hourlyQuantity = int(*r.Count)
	}

	var monthlyCPUCreditHrs *decimal.Decimal
	if r.MonthlyCPUCreditHrs != nil {
		monthlyCPUCreditHrs = decimalPtr(decimal.NewFromInt(*r.MonthlyCPUCreditHrs))
	}

	costComponents := []*engine.LineItem{
		r.dbInstanceCostComponent(hourlyQuantity),
	}

	if instanceFamily := getBurstableInstanceFamily([]string{"db.t3", "db.t4g"}, r.InstanceClass); instanceFamily != "" {
		costComponents = append(costComponents, r.cpuCreditsCostComponent(monthlyCPUCreditHrs, instanceFamily))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *NeptuneClusterInstance) dbInstanceCostComponent(quantity int) *engine.LineItem {
	usageTypePrefix := "InstanceUsage:"
	if r.IOOptimized {
		usageTypePrefix = "InstanceUsageIOOptimized:"
	}

	return &engine.LineItem{
		Name:           fmt.Sprintf("Database instance (on-demand, %s)", r.InstanceClass),
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(int64(quantity))),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("AmazonNeptune"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "instanceType", Value: strPtr(strings.ToLower(r.InstanceClass))},
				{Key: "usagetype", ValueRegex: regexPtr(fmt.Sprintf("%s%s$", usageTypePrefix, strings.ToLower(r.InstanceClass)))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}
}

func (r *NeptuneClusterInstance) cpuCreditsCostComponent(quantity *decimal.Decimal, instanceFamily string) *engine.LineItem {
	return &engine.LineItem{

		Name:           "CPU credits",
		Unit:           "vCPU-hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("AmazonNeptune"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("CPUCredits:" + instanceFamily + "$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}
