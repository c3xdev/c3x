package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAppServiceCertificateBindingRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_app_service_certificate_binding",
		CoreRFunc: NewAppServiceCertificateBinding,
		ReferenceAttributes: []string{
			"certificate_id",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"certificate_id"})
		},
	}
}
func NewAppServiceCertificateBinding(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.AppServiceCertificateBinding{Address: d.Address, Region: d.Region, SSLState: d.Get("ssl_state").String()}
	return r
}
