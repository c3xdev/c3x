package azure

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// MonitorDataCollectionRule struct represents an Azure Monitor Data Collection Rule.
//
// Resource information: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/monitor_data_collection_rule
// Pricing information: https://azure.microsoft.com/en-in/pricing/details/monitor/
type MonitorDataCollectionRule struct {
	Address string
	Region  string

	MonthlyCustomMetricsSamplesGB *int64 `c3x_usage:"monthly_custom_metrics_samples"`
}

// CoreType returns the name of this resource type
func (r *MonitorDataCollectionRule) CoreType() string {
	return "MonitorDataCollectionRule"
}

// UsageSchema defines a list which represents the usage schema of MonitorDataCollectionRule.
func (r *MonitorDataCollectionRule) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_custom_metrics_samples", ValueType: engine.Int64, DefaultValue: 0},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the MonitorDataCollectionRule.
// It uses the `c3x_usage` struct tags to populate data into the MonitorDataCollectionRule.
func (r *MonitorDataCollectionRule) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid MonitorDataCollectionRule struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *MonitorDataCollectionRule) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.metricsIngestionCostComponent(r.MonthlyCustomMetricsSamplesGB),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *MonitorDataCollectionRule) metricsIngestionCostComponent(quantity *int64) *engine.LineItem {
	var q *decimal.Decimal
	if quantity != nil {
		q = decimalPtr(decimal.NewFromInt(*quantity).Div(decimal.NewFromInt(10000000)))
	}

	return &engine.LineItem{
		Name:            "Metrics ingestion",
		Unit:            "10M samples",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: q,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Azure Monitor"),
			ProductFamily: strPtr("Management and Governance"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr("Metrics ingestion")},
				{Key: "meterName", Value: strPtr("Metrics ingestion Metric samples")},
			},
		},
		UsageBased: true,
	}
}
