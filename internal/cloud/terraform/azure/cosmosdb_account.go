package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

// This is a free resource but needs it's own custom registry item to specify the custom ID lookup function.
func getCosmosDBAccountRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:    "azurerm_cosmosdb_account",
		NoPrice: true,
		Notes:   []string{"Free resource."},

		CustomRefIDFunc: func(d *engine.ResourceSpec) []string {
			return []string{d.Get("name").String()}
		},
	}
}
