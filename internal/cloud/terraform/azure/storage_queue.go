package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getStorageQueueRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_storage_queue",
		CoreRFunc: newStorageQueue,
		ReferenceAttributes: []string{
			"storage_account_name",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"storage_account_name"})
		},
	}
}

func newStorageQueue(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	accountReplicationType := "LRS"
	accountKind := "StorageV2"

	if len(d.References("storage_account_name")) > 0 {
		storageAccount := d.References("storage_account_name")[0]

		accountTier := storageAccount.Get("account_tier").String()
		if strings.EqualFold(accountTier, "premium") {
			logging.Logger.Warn().Msgf("Skipping resource %s. Storage Queues don't support %s tier", d.Address, accountTier)
			return nil
		}

		accountReplicationType = storageAccount.Get("account_replication_type").String()
		accountKind = storageAccount.Get("account_kind").String()
	}

	switch strings.ToLower(accountReplicationType) {
	case "ragrs":
		accountReplicationType = "RA-GRS"
	case "ragzrs":
		accountReplicationType = "RA-GZRS"
	}

	return &azure.StorageQueue{
		Address:                d.Address,
		Region:                 region,
		AccountKind:            accountKind,
		AccountReplicationType: accountReplicationType,
	}
}
