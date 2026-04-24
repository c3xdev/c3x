package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

var (
	functionAppSkuMapCPU = map[string]int64{
		"ep1": 1,
		"ep2": 2,
		"ep3": 4,
	}

	functionAppSkuMapMem = map[string]float64{
		"ep1": 3.5,
		"ep2": 7.0,
		"ep3": 14.0,
	}
)

// FunctionApp struct a serverless function running in an app service environment. The billing for this
// function lies within Azure App Service, however we capture the costs in this component to make it more understandable.
//
// Resource information: https://learn.microsoft.com/en-us/azure/azure-functions/functions-overview
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/app-service/windows/
type FunctionApp struct {
	Address string
	Region  string

	SKUName string
	Tier    string
	OSType  string

	MonthlyExecutions   *int64 `c3x_usage:"monthly_executions"`
	ExecutionDurationMs *int64 `c3x_usage:"execution_duration_ms"`
	MemoryMb            *int64 `c3x_usage:"memory_mb"`
	Instances           *int64 `c3x_usage:"instances"`
}

func (r *FunctionApp) CoreType() string {
	return "FunctionApp"
}

func (r *FunctionApp) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_executions", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "execution_duration_ms", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "memory_mb", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "instances", ValueType: engine.Int64, DefaultValue: 0},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the FunctionApp struct
// It uses the `c3x_usage` struct tags to populate data into the FunctionApp
func (r *FunctionApp) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid FunctionApp struct.
//
// FunctionApp costs are CPU and Memory usage. These values rely on the user defining their expected
// usage in the usage file.
//
// Function apps are billed in two modes - Premium or Consumption.
func (r *FunctionApp) BuildResource() *engine.Estimate {
	var costComponents []*engine.LineItem

	if r.Tier == "premium" {
		cpu := r.appFunctionPremiumCPUCostComponent()
		if cpu != nil {
			costComponents = append(costComponents, cpu)
		}

		mem := r.appFunctionPremiumMemoryCostComponent()
		if mem != nil {
			costComponents = append(costComponents, mem)
		}

		return &engine.Estimate{
			Name:           r.Address,
			CostComponents: costComponents,
			UsageSchema:    r.UsageSchema(),
		}
	}

	costComponents = append(
		costComponents,
		r.appFunctionConsumptionExecutionTimeCostComponent(),
		r.appFunctionConsumptionExecutionsCostComponent(),
	)

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *FunctionApp) appFunctionPremiumCPUCostComponent() *engine.LineItem {
	var skuCPU *int64

	if val, ok := functionAppSkuMapCPU[r.SKUName]; ok {
		skuCPU = &val
	}

	if skuCPU == nil {
		return nil
	}

	instances := decimal.NewFromInt(1)
	if r.Instances != nil {
		instances = decimal.NewFromInt(*r.Instances)
	}

	return &engine.LineItem{
		Name:           fmt.Sprintf("vCPU (%s)", strings.ToUpper(r.SKUName)),
		Unit:           "vCPU",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: decimalPtr(instances.Mul(decimal.NewFromInt(*skuCPU))),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Functions"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr("vCPU Duration$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func (r *FunctionApp) appFunctionPremiumMemoryCostComponent() *engine.LineItem {
	var skuMemory *float64

	if val, ok := functionAppSkuMapMem[r.SKUName]; ok {
		skuMemory = &val
	}

	if skuMemory == nil {
		return nil
	}

	instances := decimal.NewFromInt(1)
	if r.Instances != nil {
		instances = decimal.NewFromInt(*r.Instances)
	}

	return &engine.LineItem{
		Name:           fmt.Sprintf("Memory (%s)", strings.ToUpper(r.SKUName)),
		Unit:           "GB",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: decimalPtr(instances.Mul(decimal.NewFromFloat(*skuMemory))),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Functions"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr("Memory Duration$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func (r *FunctionApp) appFunctionConsumptionExecutionTimeCostComponent() *engine.LineItem {
	gbSeconds := r.calculateFunctionAppGBSeconds()
	return &engine.LineItem{
		Name:            "Execution time",
		Unit:            "GB-seconds",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: gbSeconds,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Functions"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr("Execution Time$")},
				{Key: "skuName", Value: strPtr("Standard")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("400000"),
		},
		UsageBased: true,
	}
}

func (r *FunctionApp) appFunctionConsumptionExecutionsCostComponent() *engine.LineItem {
	// Azure's pricing API returns prices per 10 executions so if the user has provided
	// the number of executions, we should divide it by 10
	var executions *decimal.Decimal
	if r.MonthlyExecutions != nil {
		executions = decimalPtr(decimal.NewFromInt(*r.MonthlyExecutions).Div(decimal.NewFromInt(10)))
	}

	return &engine.LineItem{
		Name:            "Executions",
		Unit:            "1M requests",
		UnitMultiplier:  decimal.NewFromInt(100000),
		MonthlyQuantity: executions,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Functions"),
			ProductFamily: strPtr("Compute"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr("Total Executions$")},
				{Key: "skuName", Value: strPtr("Standard")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("100000"),
		},
		UsageBased: true,
	}
}

func (r *FunctionApp) calculateFunctionAppGBSeconds() *decimal.Decimal {
	if r.MemoryMb == nil || r.ExecutionDurationMs == nil || r.MonthlyExecutions == nil {
		return nil
	}

	memorySize := decimal.NewFromInt(*r.MemoryMb)
	averageRequestDuration := decimal.NewFromInt(*r.ExecutionDurationMs)
	monthlyRequests := decimal.NewFromInt(*r.MonthlyExecutions)

	// Use a min of 128MB, and round-up to nearest 128MB
	if memorySize.LessThan(decimal.NewFromInt(128)) {
		memorySize = decimal.NewFromInt(128)
	}
	roundedMemory := memorySize.Div(decimal.NewFromInt(128)).Ceil().Mul(decimal.NewFromInt(128))
	// Apply the minimum request duration
	if averageRequestDuration.LessThan(decimal.NewFromInt(100)) {
		averageRequestDuration = decimal.NewFromInt(100)
	}
	durationSeconds := monthlyRequests.Mul(averageRequestDuration).Mul(decimal.NewFromFloat(0.001))
	gbSeconds := durationSeconds.Mul(roundedMemory).Div(decimal.NewFromInt(1024))

	return &gbSeconds
}
