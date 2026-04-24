package azure

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMHDInsightKafkaClusterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_hdinsight_kafka_cluster", //nolint:misspell
		RFunc: NewAzureRMHDInsightKafkaCluster,
	}
}

func NewAzureRMHDInsightKafkaCluster(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region

	costComponents := []*engine.LineItem{}

	tier := d.Get("tier").String()
	diskSku := map[string]string{
		"Standard": "S30",
		"Premium":  "P30",
	}[tier]

	headNodeVM := d.Get("roles.0.head_node.0.vm_size").String()
	workerNodeVM := d.Get("roles.0.worker_node.0.vm_size").String()
	var workerInstances int64
	if d.Get("roles.0.worker_node.0.target_instance_count").Type != gjson.Null {
		workerInstances = d.Get("roles.0.worker_node.0.target_instance_count").Int()
	}
	zookeeperNodeVM := d.Get("roles.0.zookeeper_node.0.vm_size").String()

	numberOfDisks := d.Get("roles.0.worker_node.0.number_of_disks_per_node").Int()

	costComponents = append(costComponents, hdInsightVMCostComponent(region, "Head", headNodeVM, 2))
	costComponents = append(costComponents, hdInsightVMCostComponent(region, "Worker", workerNodeVM, workerInstances))
	costComponents = append(costComponents, hdInsightVMCostComponent(region, "Zookeeper", zookeeperNodeVM, 3))

	costComponents = append(costComponents, &engine.LineItem{
		Name:            "Managed OS disks",
		Unit:            "months",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(workerInstances * numberOfDisks)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("HDInsight"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(diskSku)},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Disk", diskSku))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	})

	var diskOperations *decimal.Decimal
	if u != nil && u.Get("monthly_os_disk_operations").Exists() {
		diskOperations = decimalPtr(decimal.NewFromInt(u.Get("monthly_os_disk_operations").Int()))
	}
	if diskOperations != nil {
		diskOperations = decimalPtr(diskOperations.Mul(decimal.NewFromInt(numberOfDisks * workerInstances).Div(decimal.NewFromInt(100000))))
	}
	costComponents = append(costComponents, &engine.LineItem{
		Name:            "Disk operations",
		Unit:            "100K operations",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: diskOperations,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("HDInsight"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(diskSku)},
				{Key: "meterName", Value: strPtr("Operations")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	})

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}
