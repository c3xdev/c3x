package google

import (
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getContainerNodePoolRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_container_node_pool",
		CoreRFunc: newContainerNodePool,
		ReferenceAttributes: []string{
			"cluster",
		},
		Notes: []string{
			"Sustained use discounts are applied to monthly costs, but not to hourly costs.",
			"Costs associated with non-standard Linux images, such as Windows and RHEL are not supported.",
			"Custom machine types are not supported.",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			var location string

			var cluster *engine.ResourceSpec
			if len(d.References("cluster")) > 0 {
				cluster = d.References("cluster")[0]
			}

			if cluster != nil {
				location = cluster.Get("location").String()
			}

			if d.Get("location").String() != "" {
				location = d.Get("location").String()
			}

			region := location
			if isZone(location) {
				region = zoneToRegion(location)
			}

			return region
		},
	}
}

func newContainerNodePool(d *engine.ResourceSpec) engine.CatalogItem {
	var cluster *engine.ResourceSpec
	if len(d.References("cluster")) > 0 {
		cluster = d.References("cluster")[0]
	}

	r := newNodePool(d.Address, d.RawValues, cluster)

	if r == nil {
		return nil
	}

	return r
}

func newNodePool(address string, d gjson.Result, cluster *engine.ResourceSpec) *google.ContainerNodePool {
	var location string

	if cluster != nil {
		location = cluster.Get("location").String()
	}

	if d.Get("location").String() != "" {
		location = d.Get("location").String()
	}

	region := location
	if isZone(location) {
		region = zoneToRegion(location)
	}

	if region == "" {
		logging.Logger.Warn().Msgf("Skipping resource %s. Unable to determine region", address)
		return nil
	}

	zones := int64(3)

	if cluster != nil {
		zones = int64(zoneCount(cluster.RawValues, ""))
	}

	if len(d.Get("node_locations").Array()) > 0 {
		zones = int64(zoneCount(d, location))
	}

	countPerZone := int64(3)

	if d.Get("initial_node_count").Exists() {
		countPerZone = d.Get("initial_node_count").Int()
	}

	if d.Get("node_count").Exists() {
		countPerZone = d.Get("node_count").Int()
	}

	if d.Get("autoscaling.0.min_node_count").Exists() {
		countPerZone = d.Get("autoscaling.0.min_node_count").Int()
	}

	containerNodeConfig := newContainerNodeConfig(d.Get("node_config.0"))

	return &google.ContainerNodePool{
		Address:      address,
		Region:       region,
		Zones:        zones,
		CountPerZone: countPerZone,
		NodeConfig:   containerNodeConfig,
	}
}
