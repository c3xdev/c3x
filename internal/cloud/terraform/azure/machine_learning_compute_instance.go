package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getMachineLearningComputeInstanceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_machine_learning_compute_instance",
		CoreRFunc: newMachineLearningComputeInstance,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newMachineLearningComputeInstance(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.MachineLearningComputeInstance{
		Address:      d.Address,
		Region:       region,
		InstanceType: d.Get("virtual_machine_size").String(),
	}
}
