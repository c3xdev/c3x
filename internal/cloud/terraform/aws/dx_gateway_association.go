package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDXGatewayAssociationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_dx_gateway_association",
		CoreRFunc:           NewDXGatewayAssociation,
		ReferenceAttributes: []string{"associated_gateway_id"},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			assocGateway := d.References("associated_gateway_id")
			if len(assocGateway) > 0 {
				region := assocGateway[0].Get("region").String()
				if region != "" {
					return region
				}
			}

			return defaultRegion
		},
	}
}
func NewDXGatewayAssociation(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.DXGatewayAssociation{Address: d.Address, Region: d.Get("region").String()}

	// Try to get the region from the associated gateway
	assocGateway := d.References("associated_gateway_id")
	if len(assocGateway) > 0 {
		r.AssociatedGatewayRegion = assocGateway[0].Get("region").String()
	}
	return r
}
