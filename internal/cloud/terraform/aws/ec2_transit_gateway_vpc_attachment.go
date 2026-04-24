package aws

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEC2TransitGatewayVpcAttachmentRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ec2_transit_gateway_vpc_attachment",
		CoreRFunc: NewEc2TransitGatewayVpcAttachment,
		ReferenceAttributes: []string{
			"transit_gateway_id",
			"vpc_id",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			var region string
			vpcRefs := d.References("vpc_id")
			for _, ref := range vpcRefs {
				if strings.ToLower(ref.Type) == "aws_default_vpc" || strings.ToLower(ref.Type) == "aws_vpc" {
					region = ref.Get("region").String()
					break
				}
			}

			// Try to get the region from the transit gateway
			transitGatewayRefs := d.References("transit_gateway_id")
			if len(transitGatewayRefs) > 0 {
				region = transitGatewayRefs[0].Get("region").String()
			}

			if region != "" {
				return region
			}

			return defaultRegion
		},
	}
}
func NewEc2TransitGatewayVpcAttachment(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.Ec2TransitGatewayVpcAttachment{Address: d.Address, Region: d.Get("region").String()}

	// Try to get the region from the VPC
	vpcRefs := d.References("vpc_id")
	var vpcRef *engine.ResourceSpec

	for _, ref := range vpcRefs {
		// the VPC ref can also be for the aws_subnet_ids resource which we don't want to consider
		if strings.ToLower(ref.Type) == "aws_default_vpc" || strings.ToLower(ref.Type) == "aws_vpc" {
			vpcRef = ref
			break
		}
	}
	if vpcRef != nil {
		r.VPCRegion = vpcRef.Get("region").String()
	}

	// Try to get the region from the transit gateway
	transitGatewayRefs := d.References("transit_gateway_id")
	if len(transitGatewayRefs) > 0 {
		r.TransitGatewayRegion = transitGatewayRefs[0].Get("region").String()
	}
	return r
}
