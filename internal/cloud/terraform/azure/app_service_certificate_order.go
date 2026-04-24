package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAppServiceCertificateOrderRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_app_service_certificate_order",
		CoreRFunc: NewAppServiceCertificateOrder,
	}
}
func NewAppServiceCertificateOrder(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.AppServiceCertificateOrder{Address: d.Address, ProductType: d.Get("product_type").String()}
	return r
}
