package azure

import (
	"sort"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getRecoveryServicesVaultRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_recovery_services_vault",
		CoreRFunc: newRecoveryServicesVault,
		ReferenceAttributes: []string{
			"resource_group_name",
			"azurerm_backup_protected_vm.recovery_vault_name",
		},
		CustomRefIDFunc: func(d *engine.ResourceSpec) []string {
			name := d.Get("name").String()
			if name != "" {
				return []string{name}
			}

			return nil
		},
	}
}

func newRecoveryServicesVault(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	vms := d.References("azurerm_backup_protected_vm.recovery_vault_name")

	var protectedVMs []*azure.BackupProtectedVM
	for _, vm := range vms {
		protectedVm := newBackupProtectedVm(vm)
		if protectedVm != nil {
			protectedVMs = append(protectedVMs, protectedVm)
		}
	}

	sort.Slice(protectedVMs, func(i, j int) bool {
		return protectedVMs[i].Address < protectedVMs[j].Address
	})

	return &azure.RecoveryServicesVault{
		Address:      d.Address,
		Region:       region,
		ProtectedVMs: protectedVMs,
	}
}
