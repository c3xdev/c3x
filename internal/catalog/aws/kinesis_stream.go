package aws

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// KinesisStream struct represents Kinesis Data Streams a fully managed, serverless streaming data service
//
// Resource information: https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kinesis_stream
// Pricing information: https://aws.amazon.com/kinesis/data-streams/pricing/
type KinesisStream struct {
	Address    string
	Region     string
	StreamMode string
	ShardCount int64

	// Usage fields
	// On demand
	MonthlyOnDemandDataIngestedGB      *float64 `c3x_usage:"monthly_on_demand_data_in_gb"`
	MonthlyOnDemandDataRetrievalGB     *float64 `c3x_usage:"monthly_on_demand_data_out_gb"`
	MonthlyOnDemandEFODataRetrievalGB  *float64 `c3x_usage:"monthly_on_demand_efo_data_out_gb"`
	MonthlyOnDemandExtendedRetentionGb *float64 `c3x_usage:"monthly_on_demand_extended_retention_gb"`
	MonthlyOnDemandLongTermRetentionGb *float64 `c3x_usage:"monthly_on_demand_long_term_retention_gb"`
	// Provisioned
	MonthlyProvisionedPutUnits            *float64 `c3x_usage:"monthly_provisioned_put_units"`
	MonthlyProvisionedExtendedRetentionGb *float64 `c3x_usage:"monthly_provisioned_extended_retention_gb"`
	MonthlyProvisionedLongTermRetentionGb *float64 `c3x_usage:"monthly_provisioned_long_term_retention_gb"`
	MonthlyProvisionedLongTermRetrievalGb *float64 `c3x_usage:"monthly_provisioned_long_term_retrieval_gb"`
	MonthlyProvisionedEFODataRetrievalGB  *float64 `c3x_usage:"monthly_provisioned_efo_data_out_gb"`
	MonthlyProvisionedEFOConsumerHours    *float64 `c3x_usage:"monthly_provisioned_efo_consumer_hours"`
}

// CoreType returns the name of this resource type
func (r *KinesisStream) CoreType() string {
	return "KinesisStream"
}

// UsageSchema defines a list which represents the usage schema of KinesisStream.
func (r *KinesisStream) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_on_demand_data_in_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_on_demand_data_out_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_on_demand_efo_data_out_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_on_demand_extended_retention_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_on_demand_long_term_retention_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_provisioned_put_units", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_provisioned_extended_retention_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_provisioned_long_term_retention_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_provisioned_long_term_retrieval_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_provisioned_efo_data_out_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_provisioned_efo_consumer_hours", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the KinesisStream.
// It uses the `c3x_usage` struct tags to populate data into the KinesisStream.
func (r *KinesisStream) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// Set some vars that come from the pricing api
var (
	onDemandStreamName    string = "ON_DEMAND"
	provisionedStreamName string = "PROVISIONED"
)

// BuildResource builds a engine.Estimate from a valid KinesisStream struct.
// This method is called after the resource is initialized by an IaC provider.
// See providers folder for more information.
func (r *KinesisStream) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}
	// Depending on the stream mode, we will have different cost components
	if r.StreamMode == onDemandStreamName {
		costComponents = append(costComponents, r.onDemandStreamCostComponent())
		costComponents = append(costComponents, r.onDemandDataIngestedCostComponent())
		costComponents = append(costComponents, r.onDemandDataRetrievalCostComponent())
		costComponents = append(costComponents, r.onDemandEfoDataRetrievalCostComponent())
		costComponents = append(costComponents, r.onDemandExtendedRetentionCostComponent())
		costComponents = append(costComponents, r.onDemandLongTermRetentionCostComponent())
	}
	if r.StreamMode == provisionedStreamName {
		costComponents = append(costComponents, r.provisionedStreamCostComponent())
		costComponents = append(costComponents, r.provisionedStreamPutUnitsCostComponent())
		costComponents = append(costComponents, r.provisionedExtendedRetentionCostComponent())
		costComponents = append(costComponents, r.provisionedLongTermRetentionCostComponent())
		costComponents = append(costComponents, r.provisionedLongTermRetrievalCostComponent())
		costComponents = append(costComponents, r.provisionedEfoDataRetrievalCostComponent())
		costComponents = append(costComponents, r.provisionedEfoConsumersCostComponent())
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *KinesisStream) onDemandStreamCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           onDemandStreamName,
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)OnDemand-StreamHour")},
				{Key: "operation", Value: strPtr("OnDemandStreamHr")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}
}

func (r *KinesisStream) onDemandDataIngestedCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Data ingested",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyOnDemandDataIngestedGB),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)OnDemand-BilledIncomingBytes")},
				{Key: "operation", Value: strPtr("OnDemandDataIngested")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *KinesisStream) onDemandDataRetrievalCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Data retrieval",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyOnDemandDataRetrievalGB),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)OnDemand-BilledOutgoingBytes")},
				{Key: "operation", Value: strPtr("OnDemandDataRetrieval")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *KinesisStream) onDemandEfoDataRetrievalCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Enhanced Fan Out (EFO) Data retrieval",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyOnDemandEFODataRetrievalGB),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)OnDemand-BilledOutgoingEFOBytes")},
				{Key: "operation", Value: strPtr("OnDemandEFODataRetrieval")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *KinesisStream) onDemandExtendedRetentionCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Extended retention (24H to 7D)",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyOnDemandExtendedRetentionGb),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)OnDemand-ExtendedRetention-ByteHrs")},
				{Key: "operation", Value: strPtr("OnDemandExtendedRetentionByteHrs")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *KinesisStream) onDemandLongTermRetentionCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Long term retention (7D+)",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyOnDemandLongTermRetentionGb),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)OnDemand-LongTermRetention-ByteHrs")},
				{Key: "operation", Value: strPtr("OnDemandLongTermRetentionByteHrs")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *KinesisStream) provisionedStreamCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           provisionedStreamName,
		Unit:           "hours",
		HourlyQuantity: decimalPtr(decimal.NewFromInt(r.ShardCount)),
		UnitMultiplier: decimal.NewFromInt(1),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)Storage-ShardHour")},
				{Key: "operation", Value: strPtr("shardHourStorage")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}
}

func (r *KinesisStream) provisionedStreamPutUnitsCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Put request unit",
		Unit:            "units",
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyProvisionedPutUnits),
		UnitMultiplier:  decimal.NewFromInt(1),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)PutRequestPayloadUnits")},
				{Key: "operation", Value: strPtr("PutRequest")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *KinesisStream) provisionedExtendedRetentionCostComponent() *engine.LineItem {
	return &engine.LineItem{

		Name:            "Extended retention (24H to 7D)",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyProvisionedExtendedRetentionGb),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)Extended-ShardHour")},
				{Key: "operation", Value: strPtr("shardHourStorage")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *KinesisStream) provisionedLongTermRetentionCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Long term retention (7D+)",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyProvisionedLongTermRetentionGb),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)LongTermRetention-ByteHrs")},
				{Key: "operation", Value: strPtr("LongTermRetentionByteHrs")},
			},
		},
		UsageBased: true,
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
	}
}

func (r *KinesisStream) provisionedLongTermRetrievalCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Extended retention retrieval (7D+)",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyProvisionedLongTermRetrievalGb),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)LongTermRetention-ReadBytes")},
				{Key: "operation", Value: strPtr("LongTermRetentionDataRetrieval")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *KinesisStream) provisionedEfoDataRetrievalCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Enhanced Fan Out (EFO) Data retrieval",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyProvisionedEFODataRetrievalGB),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)ReadBytes")},
				{Key: "operation", Value: strPtr("EnhancedFanoutDataRetrieval")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *KinesisStream) provisionedEfoConsumersCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Enhanced Fan Out (EFO)",
		Unit:            "consumer-shard hour",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyProvisionedEFOConsumerHours),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesis"),
			ProductFamily: strPtr("Kinesis Streams"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)EnhancedFanoutHour")},
				{Key: "operation", Value: strPtr("ConsumerHour")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}
