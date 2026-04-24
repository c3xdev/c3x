package aws

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type DocDBClusterInstance struct {
	Address             string
	Region              string
	InstanceClass       string
	DataStorageGB       *float64 `c3x_usage:"data_storage_gb"`
	MonthlyIORequests   *int64   `c3x_usage:"monthly_io_requests"`
	MonthlyCPUCreditHrs *int64   `c3x_usage:"monthly_cpu_credit_hrs"`
}

func (r *DocDBClusterInstance) CoreType() string {
	return "DocDBClusterInstance"
}

func (r *DocDBClusterInstance) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "data_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_io_requests", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_cpu_credit_hrs", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *DocDBClusterInstance) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DocDBClusterInstance) BuildResource() *engine.Estimate {
	var storageRate *decimal.Decimal
	if r.DataStorageGB != nil {
		storageRate = decimalPtr(decimal.NewFromFloat(*r.DataStorageGB))
	}

	var ioRequests *decimal.Decimal
	if r.MonthlyIORequests != nil {
		ioRequests = decimalPtr(decimal.NewFromInt(*r.MonthlyIORequests))
	}

	var cpuCredits *decimal.Decimal
	if r.MonthlyCPUCreditHrs != nil {
		cpuCredits = decimalPtr(decimal.NewFromInt(*r.MonthlyCPUCreditHrs))
	}

	costComponents := []*engine.LineItem{
		{
			Name:           fmt.Sprintf("Database instance (%s, %s)", "on-demand", r.InstanceClass),
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonDocDB"),
				ProductFamily: strPtr("Database Instance"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "instanceType", Value: strPtr(r.InstanceClass)},
					{Key: "volumeType", Value: strPtr("General Purpose")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("on_demand"),
			},
		},
		{
			Name:            "Storage",
			Unit:            "GB",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: storageRate,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonDocDB"),
				ProductFamily: strPtr("Database Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: regexPtr("(^|-)StorageUsage$")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("on_demand"),
			},
			UsageBased: true,
		},
		{
			Name:            "I/O requests",
			Unit:            "1M requests",
			UnitMultiplier:  decimal.NewFromInt(1000000),
			MonthlyQuantity: ioRequests,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonDocDB"),
				ProductFamily: strPtr("System Operation"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: regexPtr("(^|-)StorageIOUsage$")},
				},
			},
			UsageBased: true,
		},
	}

	if instanceFamily := getBurstableInstanceFamily([]string{"db.t3", "db.t4g"}, r.InstanceClass); instanceFamily != "" {
		costComponents = append(costComponents, &engine.LineItem{
			Name:            "CPU credits",
			Unit:            "vCPU-hours",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: cpuCredits,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonDocDB"),
				ProductFamily: strPtr("CPU Credits"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: regexPtr("CPUCredits:" + instanceFamily + "$")},
				},
			},
			UsageBased: true,
		})
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
