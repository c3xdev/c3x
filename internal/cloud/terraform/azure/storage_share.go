package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getStorageShareRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_storage_share",
		CoreRFunc: newStorageShare,
		ReferenceAttributes: []string{
			"storage_account_name",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"storage_account_name"})
		},
	}
}

func newStorageShare(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	accountReplicationType := "LRS"

	accessTier := d.Get("access_tier").String()
	if accessTier == "" {
		accessTier = "TransactionOptimized"
	}
	quota := d.Get("quota").Int()

	if len(d.References("storage_account_name")) > 0 {
		storageAccount := d.References("storage_account_name")[0]
		accountKind := storageAccount.Get("account_kind").String()
		accountReplicationType = storageAccount.Get("account_replication_type").String()

		if strings.EqualFold(accessTier, "premium") && !strings.EqualFold(accountKind, "filestorage") {
			logging.Logger.Warn().Msgf("Skipping resource %s. Premium access tier is only supported for FileStorage accounts", d.Address)
			return nil
		}
	}

	return &azure.StorageShare{
		Address:                d.Address,
		Region:                 region,
		AccountReplicationType: accountReplicationType,
		AccessTier:             accessTier,
		Quota:                  quota,
	}
}
