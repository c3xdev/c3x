package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAPIGatewayRestAPIRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_api_gateway_rest_api",
		CoreRFunc: NewAPIGatewayRestAPI,
	}
}
func NewAPIGatewayRestAPI(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.APIGatewayRestAPI{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
