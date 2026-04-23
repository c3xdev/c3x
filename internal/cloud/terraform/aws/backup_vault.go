package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getBackupVaultRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_backup_vault",
		CoreRFunc: NewBackupVault,
		Notes:     []string{"AWS Storage Gateway Volume Backup prices could not be found in the AWS pricing data."},
	}
}
func NewBackupVault(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.BackupVault{Address: d.Address, Region: d.Get("region").String()}
	return r
}
