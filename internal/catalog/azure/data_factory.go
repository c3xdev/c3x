package azure

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// DataFactory struct represents Azure Data Factory resource.
//
// Resource information: https://azure.microsoft.com/en-us/services/data-factory/
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/data-factory/data-pipeline/
type DataFactory struct {
	Address string
	Region  string

	// "usage" args
	MonthlyReadWriteOperationEntities  *int64 `c3x_usage:"monthly_read_write_operation_entities"`
	MonthlyMonitoringOperationEntities *int64 `c3x_usage:"monthly_monitoring_operation_entities"`
}

func (r *DataFactory) CoreType() string {
	return "DataFactory"
}

func (r *DataFactory) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_read_write_operation_entities", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_monitoring_operation_entities", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the DataFactory.
// It uses the `c3x_usage` struct tags to populate data into the DataFactory.
func (r *DataFactory) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid DataFactory struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *DataFactory) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.readWriteOperationsCostComponent(),
		r.monitoringOperationsCostComponent(),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

// readWriteOperationsCostComponent returns a cost component for
// Data Factory's Read/Write operations.
//
// The pricing is identical for all integration runtimes.
func (r *DataFactory) readWriteOperationsCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	divider := decimal.NewFromInt(50000)

	if r.MonthlyReadWriteOperationEntities != nil {
		quantity = decimalPtr(decimal.NewFromInt(*r.MonthlyReadWriteOperationEntities).Div(divider))
	}

	return &engine.LineItem{
		Name:            "Read/Write operations",
		Unit:            "50k entities",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Data Factory v2"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr("^Cloud Read Write Operations$")},
				{Key: "skuName", ValueRegex: regexPtr("^Cloud$")},
				{Key: "productName", ValueRegex: regexPtr("^Azure Data Factory v2$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
		UsageBased: true,
	}
}

// monitoringOperationsCostComponent returns a cost component for
// Data Factory's Monitoring operations.
//
// The pricing is identical for all integration runtimes.
func (r *DataFactory) monitoringOperationsCostComponent() *engine.LineItem {
	var quantity *decimal.Decimal
	divider := decimal.NewFromInt(50000)

	if r.MonthlyMonitoringOperationEntities != nil {
		quantity = decimalPtr(decimal.NewFromInt(*r.MonthlyMonitoringOperationEntities).Div(divider))
	}

	return &engine.LineItem{
		Name:            "Monitoring operations",
		Unit:            "50k entities",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Data Factory v2"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr("^Cloud Monitoring Operations$")},
				{Key: "skuName", ValueRegex: regexPtr("^Cloud$")},
				{Key: "productName", ValueRegex: regexPtr("^Azure Data Factory v2$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
		UsageBased: true,
	}
}
