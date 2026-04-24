package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEC2ClientVPNEndpointRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ec2_client_vpn_endpoint",
		CoreRFunc: NewEc2ClientVpnEndpoint,
	}
}
func NewEc2ClientVpnEndpoint(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.EC2ClientVPNEndpoint{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
