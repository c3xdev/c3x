package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type VPNConnection struct {
	Address                string
	Region                 string
	TransitGatewayID       string
	MonthlyDataProcessedGB *float64 `c3x_usage:"monthly_data_processed_gb"`
}

func (r *VPNConnection) CoreType() string {
	return "VPNConnection"
}

func (r *VPNConnection) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_data_processed_gb", ValueType: engine.Float64, DefaultValue: 0}}
}

func (r *VPNConnection) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *VPNConnection) BuildResource() *engine.Estimate {
	region := r.Region

	var gbDataProcessed *decimal.Decimal

	costComponents := []*engine.LineItem{
		{
			Name:           "VPN connection",
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonVPC"),
				ProductFamily: strPtr("Cloud Connectivity"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "vpnType", ValueRegex: regexPtr("^VPN Standard")},
				},
			},
		},
	}

	if r.TransitGatewayID != "" {
		costComponents = append(costComponents, transitGatewayAttachmentCostComponent(region, "TransitGatewayVPN"))

		if r.MonthlyDataProcessedGB != nil {
			gbDataProcessed = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataProcessedGB))
		}

		costComponents = append(costComponents, transitGatewayDataProcessingCostComponent(region, "TransitGatewayVPN", gbDataProcessed))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
