package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/usage"
)

type KinesisFirehoseDeliveryStream struct {
	Address                     string
	Region                      string
	DataFormatConversionEnabled bool
	VPCDeliveryEnabled          bool
	VPCDeliveryAZs              int64
	MonthlyDataIngestedGB       *float64 `c3x_usage:"monthly_data_ingested_gb"`
}

func (r *KinesisFirehoseDeliveryStream) CoreType() string {
	return "KinesisFirehoseDeliveryStream"
}

func (r *KinesisFirehoseDeliveryStream) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_data_ingested_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *KinesisFirehoseDeliveryStream) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *KinesisFirehoseDeliveryStream) BuildResource() *engine.Estimate {
	costComponents := make([]*engine.LineItem, 0)

	if r.MonthlyDataIngestedGB != nil {
		tierLimits := []int{512_000, 1_536_000}

		result := usage.CalculateTierBuckets(decimal.NewFromFloat(*r.MonthlyDataIngestedGB), tierLimits)

		if result[0].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, r.dataIngestedCostComponent("first 500TB", "0", "512000", &result[0]))
		}
		if result[1].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, r.dataIngestedCostComponent("next 1.5PB", "512000", "2048000", &result[1]))
		}
		if result[2].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, r.dataIngestedCostComponent("next 3PB", "2048000", "Inf", &result[2]))
		}
	} else {
		costComponents = append(costComponents, r.dataIngestedCostComponent("first 500TB", "0", "512000", nil))
	}

	if r.DataFormatConversionEnabled {
		costComponents = append(costComponents, r.formatConversionCostComponent())
	}

	if r.VPCDeliveryEnabled {
		costComponents = append(costComponents, r.vpcDataCostComponent())
		costComponents = append(costComponents, r.vpcDeliveryCostComponent())
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *KinesisFirehoseDeliveryStream) dataIngestedCostComponent(tier, startUsageAmount, endUsageAmount string, quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            fmt.Sprintf("Data ingested (%s)", tier),
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesisFirehose"),
			ProductFamily: strPtr("Kinesis Firehose"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "group", Value: strPtr("Event-by-Event Processing")},
				{Key: "sourcetype", Value: strPtr("")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(startUsageAmount),
			EndUsageAmount:   strPtr(endUsageAmount),
		},
		UsageBased: true,
	}
}

func (r *KinesisFirehoseDeliveryStream) formatConversionCostComponent() *engine.LineItem {
	var monthlyDataIngestedGB *decimal.Decimal
	if r.MonthlyDataIngestedGB != nil {
		monthlyDataIngestedGB = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataIngestedGB))
	}

	return &engine.LineItem{
		Name:            "Format conversion",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyDataIngestedGB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesisFirehose"),
			ProductFamily: strPtr("Kinesis Firehose"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "operation", Value: strPtr("DataFormatConversion")},
			},
		},
	}
}

func (r *KinesisFirehoseDeliveryStream) vpcDataCostComponent() *engine.LineItem {
	var monthlyDataIngestedGB *decimal.Decimal
	if r.MonthlyDataIngestedGB != nil {
		monthlyDataIngestedGB = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataIngestedGB))
	}

	return &engine.LineItem{
		Name:            "VPC data",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyDataIngestedGB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesisFirehose"),
			ProductFamily: strPtr("Kinesis Firehose"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "operation", Value: strPtr("VpcBandwidth")},
			},
		},
	}
}

func (r *KinesisFirehoseDeliveryStream) vpcDeliveryCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           "VPC AZ delivery",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(r.VPCDeliveryAZs)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesisFirehose"),
			ProductFamily: strPtr("Kinesis Firehose"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "operation", Value: strPtr("RunVpcInstance")},
			},
		},
	}
}
