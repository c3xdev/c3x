package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEC2ClientVPNNetworkAssociationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ec2_client_vpn_network_association",
		CoreRFunc: NewEC2ClientVPNNetworkAssociation,
	}
}
func NewEC2ClientVPNNetworkAssociation(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.EC2ClientVPNNetworkAssociation{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
