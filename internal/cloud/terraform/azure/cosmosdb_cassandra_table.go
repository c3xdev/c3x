package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

func GetAzureRMCosmosdbCassandraTableRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_cosmosdb_cassandra_table",
		RFunc: NewAzureRMCosmosdbCassandraTable,
		ReferenceAttributes: []string{
			"account_name",
			"cassandra_keyspace_id",
		},
	}
}

func NewAzureRMCosmosdbCassandraTable(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	if len(d.References("cassandra_keyspace_id")) > 0 {
		keyspace := d.References("cassandra_keyspace_id")[0]
		if len(keyspace.References("account_name")) > 0 {
			account := keyspace.References("account_name")[0]
			return &engine.Estimate{
				Name:           d.Address,
				CostComponents: cosmosDBCostComponents(d, u, account),
			}
		}
		logging.Logger.Warn().Msgf("Skipping resource %s as its 'cassandra_keyspace_id.account_name' property could not be found.", d.Address)
		return nil
	}
	logging.Logger.Warn().Msgf("Skipping resource %s as its 'cassandra_keyspace_id' property could not be found.", d.Address)
	return nil
}
