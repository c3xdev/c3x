package azure

import (
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMHDInsightHBaseClusterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_hdinsight_hbase_cluster", //nolint:misspell
		RFunc: NewAzureRMHDInsightHBaseCluster,
	}
}

func NewAzureRMHDInsightHBaseCluster(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region

	costComponents := []*engine.LineItem{}

	headNodeVM := d.Get("roles.0.head_node.0.vm_size").String()
	regionNodeVM := d.Get("roles.0.worker_node.0.vm_size").String()
	var regionInstances int64
	if d.Get("roles.0.worker_node.0.target_instance_count").Type != gjson.Null {
		regionInstances = d.Get("roles.0.worker_node.0.target_instance_count").Int()
	}
	zookeeperNodeVM := d.Get("roles.0.zookeeper_node.0.vm_size").String()

	costComponents = append(costComponents, hdInsightVMCostComponent(region, "Head", headNodeVM, 2))
	costComponents = append(costComponents, hdInsightVMCostComponent(region, "Region", regionNodeVM, regionInstances))
	costComponents = append(costComponents, hdInsightVMCostComponent(region, "Zookeeper", zookeeperNodeVM, 3))

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}
