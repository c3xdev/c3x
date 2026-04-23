package azure

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// DataFactoryIntegrationRuntimeAzure struct represents Azure Data Factory's
// runtime flow.
//
// Resource information: https://azure.microsoft.com/en-us/services/data-factory/
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/data-factory/data-pipeline/
type DataFactoryIntegrationRuntimeAzure struct {
	Address string
	Region  string

	Cores       int64
	ComputeType string

	// "usage" args
	MonthlyOrchestrationRuns *int64 `c3x_usage:"monthly_orchestration_runs"`
}

func (r *DataFactoryIntegrationRuntimeAzure) CoreType() string {
	return "DataFactoryIntegrationRuntimeAzure"
}

func (r *DataFactoryIntegrationRuntimeAzure) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_orchestration_runs", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the DataFactoryIntegrationRuntimeAzure.
// It uses the `c3x_usage` struct tags to populate data into the DataFactoryIntegrationRuntimeAzure.
func (r *DataFactoryIntegrationRuntimeAzure) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid DataFactoryIntegrationRuntimeAzure struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *DataFactoryIntegrationRuntimeAzure) BuildResource() *engine.Estimate {
	runtimeFilter := "Cloud"

	costComponents := []*engine.LineItem{
		r.computeCostComponent(),
		dataFactoryOrchestrationCostComponent(r.Region, runtimeFilter, r.MonthlyOrchestrationRuns),
		dataFactoryDataMovementCostComponent(r.Region, runtimeFilter),
		dataFactoryPipelineCostComponent(r.Region, runtimeFilter),
		dataFactoryExternalPipelineCostComponent(r.Region, runtimeFilter),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

// computeCostComponent returns a cost component for cluster configuration.
//
// CPAPI contains 2 records with the same price, but one is newer and its
// armSkuName is not empty thus using a non-empty filter.
func (r *DataFactoryIntegrationRuntimeAzure) computeCostComponent() *engine.LineItem {

	productType := map[string]string{
		"general":           "General Purpose",
		"compute_optimized": "Compute Optimized",
		"memory_optimized":  "Memory Optimized",
	}[r.ComputeType]

	name := fmt.Sprintf("Compute (%s, %d vCores)", productType, r.Cores)

	return &engine.LineItem{
		Name:           name,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(r.Cores),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(r.Cores)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Data Factory v2"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", ValueRegex: regexPtr("^vCore$")},
				{Key: "productName", ValueRegex: regexPtr(fmt.Sprintf("^Azure Data Factory v2 Data Flow - %s$", productType))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func dataFactoryOrchestrationCostComponent(region string, filter string, runs *int64) *engine.LineItem {
	var quantity *decimal.Decimal
	divider := decimal.NewFromInt(1000)

	if runs != nil {
		quantity = decimalPtr(decimal.NewFromInt(*runs).Div(divider))
	}

	return &engine.LineItem{
		Name:            "Orchestration",
		Unit:            "1k runs",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Azure Data Factory v2"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("^%s Orchestration Activity Run$", filter))},
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", filter))},
				{Key: "productName", ValueRegex: regexPtr("^Azure Data Factory v2$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
		UsageBased: true,
	}
}

func dataFactoryDataMovementCostComponent(region string, filter string) *engine.LineItem {
	return &engine.LineItem{
		Name:           "Data movement activity",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Azure Data Factory v2"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("^%s Data Movement$", filter))},
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", filter))},
				{Key: "productName", ValueRegex: regexPtr("^Azure Data Factory v2$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

// dataFactoryPipelineCostComponent returns a cost component for Data Factory
// runtime's pipeline activity.
func dataFactoryPipelineCostComponent(region string, filter string) *engine.LineItem {
	return &engine.LineItem{
		Name:           "Pipeline activity",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Azure Data Factory v2"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("^%s Pipeline Activity$", filter))},
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", filter))},
				{Key: "productName", ValueRegex: regexPtr("^Azure Data Factory v2$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

// dataFactoryExternalPipelineCostComponent returns a cost component for Data
// Factory runtime's external pipeline activity.
func dataFactoryExternalPipelineCostComponent(region string, filter string) *engine.LineItem {
	return &engine.LineItem{
		Name:           "External pipeline activity",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Azure Data Factory v2"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("^%s External Pipeline Activity$", filter))},
				{Key: "skuName", ValueRegex: regexPtr(fmt.Sprintf("^%s$", filter))},
				{Key: "productName", ValueRegex: regexPtr("^Azure Data Factory v2$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
