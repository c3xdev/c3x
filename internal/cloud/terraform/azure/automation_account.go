package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAutomationAccountRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_automation_account",
		CoreRFunc: NewAutomationAccount,
	}
}
func NewAutomationAccount(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.AutomationAccount{Address: d.Address, Region: d.Region}
	return r
}
