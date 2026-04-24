package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAzureRMMSSQLManagedInstanceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_mssql_managed_instance",
		CoreRFunc: newMSSQLManagedInstance,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newMSSQLManagedInstance(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	r := &azure.MSSQLManagedInstance{
		Address: d.Address,
		Region:  region,
	}

	r.SKU = d.Get("sku_name").String()
	r.Cores = d.Get("vcores").Int()
	r.LicenseType = d.Get("license_type").String()
	r.StorageAccountType = d.Get("storage_account_type").String()
	if r.StorageAccountType == "" {
		r.StorageAccountType = "LRS"
	}
	if r.StorageAccountType == "GRS" {
		r.StorageAccountType = "RA-GRS"
	}
	r.StorageSizeInGb = d.Get("storage_size_in_gb").Int()

	return r
}
