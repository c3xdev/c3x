package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getStorageManagementPolicyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_storage_management_policy",
		RFunc: newStorageManagementPolicy,
		ReferenceAttributes: []string{
			"storage_account_id",
		},
	}
}

func newStorageManagementPolicy(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	return &engine.Estimate{
		Name:      d.Address,
		NoPrice:   true,
		IsSkipped: true,
	}
}
