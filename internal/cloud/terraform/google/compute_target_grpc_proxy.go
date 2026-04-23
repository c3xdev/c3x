package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeTargetGRPCProxyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_target_grpc_proxy",
		CoreRFunc: NewComputeTargetGRPCProxy,
		Notes:     []string{"Price for additional forwarding rule is used"},
	}
}
func getComputeTargetHTTPProxyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_target_http_proxy",
		CoreRFunc: NewComputeTargetGRPCProxy,
		Notes:     []string{"Price for additional forwarding rule is used"},
	}
}
func getComputeTargetHTTPSProxyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_target_https_proxy",
		CoreRFunc: NewComputeTargetGRPCProxy,
		Notes:     []string{"Price for additional forwarding rule is used"},
	}
}
func getComputeTargetSSLProxyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_target_ssl_proxy",
		CoreRFunc: NewComputeTargetGRPCProxy,
		Notes:     []string{"Price for additional forwarding rule is used"},
	}
}
func getComputeTargetTCPProxyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_target_tcp_proxy",
		CoreRFunc: NewComputeTargetGRPCProxy,
		Notes:     []string{"Price for additional forwarding rule is used"},
	}
}
func getComputeRegionTargetHTTPProxyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_region_target_http_proxy",
		CoreRFunc: NewComputeTargetGRPCProxy,
		Notes:     []string{"Price for additional forwarding rule is used"},
	}
}
func getComputeRegionTargetHTTPSProxyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_region_target_https_proxy",
		CoreRFunc: NewComputeTargetGRPCProxy,
		Notes:     []string{"Price for additional forwarding rule is used"},
	}
}

func NewComputeTargetGRPCProxy(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.ComputeTargetGRPCProxy{Address: d.Address, Region: d.Get("region").String()}
	return r
}
