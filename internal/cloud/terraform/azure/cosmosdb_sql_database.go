package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMCosmosdbSQLDatabaseRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_cosmosdb_sql_database",
		RFunc: NewAzureRMCosmosdb,
		ReferenceAttributes: []string{
			"account_name",
			"resource_group_name",
		},
	}
}
