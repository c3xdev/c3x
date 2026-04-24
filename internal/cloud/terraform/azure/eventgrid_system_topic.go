package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEventgridSystemTopicRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "azurerm_eventgrid_system_topic",
		CoreRFunc: func(d *engine.ResourceSpec) engine.CatalogItem {
			return &azure.EventGridTopic{
				Address: d.Address,
				Region:  d.Region,
			}
		},
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
