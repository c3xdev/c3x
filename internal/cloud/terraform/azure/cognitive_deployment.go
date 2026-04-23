package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCognitiveDeploymentRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_cognitive_deployment",
		CoreRFunc: newCognitiveDeployment,
		ReferenceAttributes: []string{
			"cognitive_account_id",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			region := lookupRegion(d, []string{"cognitive_account_id"})

			cognitiveAccountRefs := d.References("cognitive_account_id")
			if region == "" && len(cognitiveAccountRefs) > 0 {
				region = lookupRegion(cognitiveAccountRefs[0], []string{"resource_group_name"})
			}

			return region
		},
	}
}

func newCognitiveDeployment(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	return &azure.CognitiveDeployment{
		Address:  d.Address,
		Region:   region,
		Model:    strings.ToLower(d.Get("model.0.name").String()),
		Version:  strings.ToLower(d.Get("model.0.version").String()),
		Tier:     strings.ToLower(d.Get("scale.0.tier").String()),
		SKU:      strings.ToLower(d.Get("sku.0.name").String()),
		Capacity: d.Get("sku.0.capacity").Int(),
	}
}
