package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type DXGatewayAssociation struct {
	Address                 string
	Region                  string
	AssociatedGatewayRegion string
	MonthlyDataProcessedGB  *float64 `c3x_usage:"monthly_data_processed_gb"`
}

func (r *DXGatewayAssociation) CoreType() string {
	return "DXGatewayAssociation"
}

func (r *DXGatewayAssociation) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_data_processed_gb", ValueType: engine.Float64, DefaultValue: 0}}
}

func (r *DXGatewayAssociation) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DXGatewayAssociation) BuildResource() *engine.Estimate {
	region := r.Region

	if r.AssociatedGatewayRegion != "" {
		region = r.AssociatedGatewayRegion
	}

	var gbDataProcessed *decimal.Decimal

	if r.MonthlyDataProcessedGB != nil {
		gbDataProcessed = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataProcessedGB))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			transitGatewayDataProcessingCostComponent(region, "TransitGatewayDirectConnect", gbDataProcessed),
			transitGatewayAttachmentCostComponent(region, "TransitGatewayDirectConnect"),
		}, UsageSchema: r.UsageSchema(),
	}
}
