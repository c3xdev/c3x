package google

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getServiceNetworkingConnectionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "google_service_networking_connection",
		RFunc: newServiceNetworkingConnection,
	}
}

func newServiceNetworkingConnection(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Get("region").String()
	return &engine.Estimate{
		Name: d.Address,
		SubResources: []*engine.Estimate{
			networkEgress(region, u, "Network egress", "Traffic", ComputeVPNGateway),
		},
	}
}
