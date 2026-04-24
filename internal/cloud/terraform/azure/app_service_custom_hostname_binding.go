package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAppServiceCustomHostnameBindingRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_app_service_custom_hostname_binding",
		CoreRFunc: NewAppServiceCustomHostnameBinding,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewAppServiceCustomHostnameBinding(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.AppServiceCustomHostnameBinding{Address: d.Address, SSLState: d.Get("ssl_state").String()}
	r.Region = "Global"
	group := d.References("resource_group_name")
	if len(group) > 0 {
		r.Region = group[0].Get("location").String()
	}
	return r
}
