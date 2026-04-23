package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getMachineLearningComputeClusterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_machine_learning_compute_cluster",
		CoreRFunc: newMachineLearningComputeCluster,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newMachineLearningComputeCluster(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.MachineLearningComputeCluster{
		Address:      d.Address,
		Region:       region,
		InstanceType: d.Get("vm_size").String(),
		MinNodeCount: d.Get("scale_settings.0.min_node_count").Int(),
	}
}
