package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAPIGatewayV2APIRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_apigatewayv2_api",
		CoreRFunc: NewAPIGatewayV2API,
	}
}
func NewAPIGatewayV2API(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.APIGatewayV2API{
		Address:      d.Address,
		ProtocolType: d.Get("protocol_type").String(),
		Region:       d.Get("region").String(),
	}
	return r
}
