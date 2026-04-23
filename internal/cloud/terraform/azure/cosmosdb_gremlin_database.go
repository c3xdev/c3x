package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMCosmosdbGremlinDatabaseRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_cosmosdb_gremlin_database",
		RFunc: NewAzureRMCosmosdb,
		ReferenceAttributes: []string{
			"account_name",
			"resource_group_name",
		},
	}
}
