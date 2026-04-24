package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEC2TransitGatewayPeeringAttachmentRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ec2_transit_gateway_peering_attachment",
		CoreRFunc: NewEC2TransitGatewayPeeringAttachment,
		ReferenceAttributes: []string{
			"transit_gateway_id",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			transitGatewayRefs := d.References("transit_gateway_id")
			if len(transitGatewayRefs) > 0 {
				region := transitGatewayRefs[0].Get("region").String()
				if region != "" {
					return region
				}
			}

			return defaultRegion
		},
	}
}
func NewEC2TransitGatewayPeeringAttachment(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.EC2TransitGatewayPeeringAttachment{Address: d.Address, Region: d.Get("region").String()}

	transitGatewayRefs := d.References("transit_gateway_id")
	if len(transitGatewayRefs) > 0 {
		r.TransitGatewayRegion = transitGatewayRefs[0].Get("region").String()
	}
	return r
}
