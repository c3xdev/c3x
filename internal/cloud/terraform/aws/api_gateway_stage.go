package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAPIGatewayStageRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_api_gateway_stage",
		CoreRFunc: NewAPIGatewayStage,
	}
}
func NewAPIGatewayStage(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.APIGatewayStage{
		Address:          d.Address,
		Region:           d.Get("region").String(),
		CacheClusterSize: d.Get("cache_cluster_size").Float(),
		CacheEnabled:     d.GetBoolOrDefault("cache_cluster_enabled", false),
	}
	return r
}
