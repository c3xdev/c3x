package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type Ec2TransitGatewayVpcAttachment struct {
	Address                string
	Region                 string
	VPCRegion              string
	TransitGatewayRegion   string
	MonthlyDataProcessedGB *float64 `c3x_usage:"monthly_data_processed_gb"`
}

func (r *Ec2TransitGatewayVpcAttachment) CoreType() string {
	return "Ec2TransitGatewayVpcAttachment"
}

func (r *Ec2TransitGatewayVpcAttachment) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_data_processed_gb", ValueType: engine.Float64, DefaultValue: 0}}
}

func (r *Ec2TransitGatewayVpcAttachment) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *Ec2TransitGatewayVpcAttachment) BuildResource() *engine.Estimate {
	region := r.Region

	if r.VPCRegion != "" {
		region = r.VPCRegion
	}

	if r.TransitGatewayRegion != "" {
		region = r.TransitGatewayRegion
	}

	var gbDataProcessed *decimal.Decimal

	if r.MonthlyDataProcessedGB != nil {
		gbDataProcessed = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataProcessedGB))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			transitGatewayAttachmentCostComponent(region, "TransitGatewayVPC"),
			transitGatewayDataProcessingCostComponent(region, "TransitGatewayVPC", gbDataProcessed),
		}, UsageSchema: r.UsageSchema(),
	}
}

func transitGatewayAttachmentCostComponent(region string, operation string) *engine.LineItem {
	return &engine.LineItem{
		Name:           "Transit gateway attachment",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr("AmazonVPC"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/TransitGateway-Hours/")},
				{Key: "operation", Value: strPtr(operation)},
			},
		},
	}
}

func transitGatewayDataProcessingCostComponent(region string, operation string, gbDataProcessed *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Data processed",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: gbDataProcessed,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr("AmazonVPC"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/TransitGateway-Bytes/")},
				{Key: "operation", Value: strPtr(operation)},
			},
		},
		UsageBased: true,
	}
}
