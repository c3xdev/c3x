package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getVPNConnectionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_vpn_connection",
		CoreRFunc: NewVPNConnection,
	}
}
func NewVPNConnection(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.VPNConnection{Address: d.Address, TransitGatewayID: d.Get("transit_gateway_id").String(), Region: d.Get("region").String()}
	return r
}
