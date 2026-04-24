package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type CloudwatchEventBus struct {
	Address                      string
	Region                       string
	MonthlySchemaDiscoveryEvents *int64   `c3x_usage:"monthly_schema_discovery_events"`
	MonthlyCustomEvents          *int64   `c3x_usage:"monthly_custom_events"`
	MonthlyThirdPartyEvents      *int64   `c3x_usage:"monthly_third_party_events"`
	MonthlyArchiveProcessingGB   *float64 `c3x_usage:"monthly_archive_processing_gb"`
	ArchiveStorageGB             *float64 `c3x_usage:"archive_storage_gb"`
}

func (r *CloudwatchEventBus) CoreType() string {
	return "CloudwatchEventBus"
}

func (r *CloudwatchEventBus) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_schema_discovery_events", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_custom_events", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_third_party_events", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_archive_processing_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "archive_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *CloudwatchEventBus) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *CloudwatchEventBus) BuildResource() *engine.Estimate {
	var monthlyCustomEvents *decimal.Decimal
	if r.MonthlyCustomEvents != nil {
		monthlyCustomEvents = decimalPtr(decimal.NewFromInt(*r.MonthlyCustomEvents))
	}
	var monthlyPartnerEvents *decimal.Decimal
	if r.MonthlyThirdPartyEvents != nil {
		monthlyPartnerEvents = decimalPtr(decimal.NewFromInt(*r.MonthlyThirdPartyEvents))
	}
	var monthlyArchiveProcessing *decimal.Decimal
	if r.MonthlyArchiveProcessingGB != nil {
		monthlyArchiveProcessing = decimalPtr(decimal.NewFromFloat(*r.MonthlyArchiveProcessingGB))
	}
	var monthlyArchivedEvents *decimal.Decimal
	if r.ArchiveStorageGB != nil {
		monthlyArchivedEvents = decimalPtr(decimal.NewFromFloat(*r.ArchiveStorageGB))
	}
	var monthlyIngestedEvents *decimal.Decimal
	if r.MonthlySchemaDiscoveryEvents != nil {
		monthlyIngestedEvents = decimalPtr(decimal.NewFromInt(*r.MonthlySchemaDiscoveryEvents))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Custom events published",
				Unit:            "1M events",
				UnitMultiplier:  decimal.NewFromInt(1000000),
				MonthlyQuantity: monthlyCustomEvents,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSEvents"),
					ProductFamily: strPtr("EventBridge"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "eventType", Value: strPtr("Custom Event")},
						{Key: "usagetype", ValueRegex: strPtr("/Event-64K-Chunks/")},
					},
				},
				UsageBased: true,
			},
			{
				Name:            "Third-party events published",
				Unit:            "1M events",
				UnitMultiplier:  decimal.NewFromInt(1000000),
				MonthlyQuantity: monthlyPartnerEvents,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSEvents"),
					ProductFamily: strPtr("EventBridge"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "eventType", Value: strPtr("Partner Event")},
						{Key: "usagetype", ValueRegex: strPtr("/Event-64K-Chunks/")},
					},
				},
				UsageBased: true,
			},
			{
				Name:            "Archive processing",
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: monthlyArchiveProcessing,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSEvents"),
					ProductFamily: strPtr("CloudWatch Events"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/ArchivedEvents-Bytes/")},
					},
				},
				UsageBased: true,
			},
			{
				Name:            "Archive storage",
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: monthlyArchivedEvents,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSEvents"),
					ProductFamily: strPtr("CloudWatch Events"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/TimedStorage-ByteHrs/")},
					},
				},
				UsageBased: true,
			},
			{
				Name:            "Schema discovery",
				Unit:            "1M events",
				UnitMultiplier:  decimal.NewFromInt(1000000),
				MonthlyQuantity: monthlyIngestedEvents,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSEvents"),
					ProductFamily: strPtr("EventBridge"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "eventType", Value: strPtr("Discovery Event")},
						{Key: "usagetype", ValueRegex: strPtr("/Event-8K-Chunks/")},
					},
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
