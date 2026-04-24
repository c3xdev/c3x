package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

func GetAzureRMCosmosdbMongoCollectionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_cosmosdb_mongo_collection",
		RFunc: NewAzureRMCosmosdbMongoCollection,
		ReferenceAttributes: []string{
			"account_name",
			"database_name",
			"resource_group_name",
		},
	}
}

func NewAzureRMCosmosdbMongoCollection(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	if len(d.References("database_name")) > 0 {
		mongoDB := d.References("database_name")[0]
		if len(mongoDB.References("account_name")) > 0 {
			account := mongoDB.References("account_name")[0]
			return &engine.Estimate{
				Name:           d.Address,
				CostComponents: cosmosDBCostComponents(d, u, account),
			}
		}
		logging.Logger.Warn().Msgf("Skipping resource %s as its 'database_name.account_name' property could not be found.", d.Address)
		return nil
	}
	logging.Logger.Warn().Msgf("Skipping resource %s as its 'database_name' property could not be found.", d.Address)
	return nil
}
