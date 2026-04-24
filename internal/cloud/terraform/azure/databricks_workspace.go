package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDatabricksWorkspaceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_databricks_workspace",
		CoreRFunc: NewDatabricksWorkspace,
	}
}
func NewDatabricksWorkspace(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DatabricksWorkspace{Address: d.Address, Region: d.Region, SKU: d.Get("sku").String()}
	return r
}
