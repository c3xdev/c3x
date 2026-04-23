package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getMonitoringItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_monitoring_metric_descriptor",
		CoreRFunc: NewMonitoringMetricDescriptor,
	}
}

func NewMonitoringMetricDescriptor(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.MonitoringMetricDescriptor{
		Address: d.Address,
	}

	return r
}
