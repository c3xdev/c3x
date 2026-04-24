package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type EC2TransitGatewayPeeringAttachment struct {
	Address              string
	Region               string
	TransitGatewayRegion string
}

func (r *EC2TransitGatewayPeeringAttachment) CoreType() string {
	return "EC2TransitGatewayPeeringAttachment"
}

func (r *EC2TransitGatewayPeeringAttachment) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *EC2TransitGatewayPeeringAttachment) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EC2TransitGatewayPeeringAttachment) BuildResource() *engine.Estimate {
	region := r.Region
	if r.TransitGatewayRegion != "" {
		region = r.TransitGatewayRegion
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			transitGatewayAttachmentCostComponent(region, "TransitGatewayPeering"),
		}, UsageSchema: r.UsageSchema(),
	}
}
