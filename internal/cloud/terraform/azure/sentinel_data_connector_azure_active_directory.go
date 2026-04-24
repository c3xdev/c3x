package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getSentinelDataConnectorAzureActiveDirectoryRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_sentinel_data_connector_azure_active_directory",
		RFunc: newSentinelDataConnectorAzureActiveDirectory,
		ReferenceAttributes: []string{
			"log_analytics_workspace_id",
		},
	}
}

func newSentinelDataConnectorAzureActiveDirectory(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	return &engine.Estimate{
		Name:        d.Address,
		IsSkipped:   true,
		NoPrice:     true,
		UsageSchema: []*engine.ConsumptionField{},
	}
}
