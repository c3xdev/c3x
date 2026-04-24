package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMCosmosdbGremlinGraphRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_cosmosdb_gremlin_graph",
		RFunc: NewAzureRMCosmosdb,
		ReferenceAttributes: []string{
			"account_name",
			"resource_group_name",
		},
	}
}
