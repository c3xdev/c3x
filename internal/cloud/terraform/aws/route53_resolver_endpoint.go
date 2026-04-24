package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getRoute53ResolverEndpointRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_route53_resolver_endpoint",
		CoreRFunc: NewRoute53ResolverEndpoint,
	}
}

func NewRoute53ResolverEndpoint(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.Route53ResolverEndpoint{
		Address:           d.Address,
		Region:            d.Get("region").String(),
		ResolverEndpoints: int64(len(d.Get("ip_address").Array())),
	}
	return r
}
