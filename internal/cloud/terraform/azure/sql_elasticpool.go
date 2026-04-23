package azure

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSQLElasticPoolRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_sql_elasticpool",
		CoreRFunc: newSQLElasticPool,
		ReferenceAttributes: []string{
			"server_name",
			"resource_group_name",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"server_name", "resource_group_name"})
		},
	}
}

func newSQLElasticPool(d *engine.ResourceSpec) engine.CatalogItem {
	tier := d.Get("edition").String()
	sku := fmt.Sprintf("%sPool", strings.ToTitle(tier))
	dtu := d.Get("dtu").Int()

	region := d.Region
	r := &azure.MSSQLElasticPool{
		Address:       d.Address,
		Region:        region,
		SKU:           sku,
		Family:        "",
		Tier:          tier,
		DTUCapacity:   &dtu,
		LicenseType:   "LicenseIncluded",
		ZoneRedundant: d.Get("zone_redundant").Bool(),
	}

	if !d.IsEmpty("pool_size") {
		maxSizeGB := d.Get("pool_size").Float() / 1024.0
		r.MaxSizeGB = &maxSizeGB
	}

	return r
}
