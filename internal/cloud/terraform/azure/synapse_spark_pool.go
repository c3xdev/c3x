package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMSynapseSparkPoolRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_synapse_spark_pool",
		RFunc: NewAzureRMSynapseSparkPool,
		ReferenceAttributes: []string{
			"synapse_workspace_id",
		},
		Notes: []string{"the total costs consist of several resources that should be viewed as a whole"},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"synapse_workspace_id"})
		},
	}
}

func NewAzureRMSynapseSparkPool(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region
	costComponents := make([]*engine.LineItem, 0)

	nodeSize := "Small"
	if d.Get("node_size").Type != gjson.Null {
		nodeSize = d.Get("node_size").String()
	}

	nodevCores := synapseSparkPoolNodeSize(nodeSize)

	var nodeCount *decimal.Decimal
	if d.Get("node_count").Type != gjson.Null {
		nodeCount = decimalPtr(decimal.NewFromInt(d.Get("node_count").Int()))
	}

	if nodeCount == nil {
		if d.Get("auto_scale").Type != gjson.Null {
			autoScale := d.Get("auto_scale").Array()
			if len(autoScale) > 0 {
				nodeCount = decimalPtr(decimal.NewFromInt(autoScale[0].Get("min_node_count").Int()))
			}
		}
	}

	var hours *decimal.Decimal
	if u != nil && u.Get("monthly_hours").Type != gjson.Null {
		hours = decimalPtr(decimal.NewFromInt(u.Get("monthly_hours").Int()))
	}

	costComponents = append(costComponents, synapseSparkPoolCostComponent(region, fmt.Sprintf("%s (%s nodes)", strings.ToLower(nodeSize), nodeCount), "120", nodeCount, nodevCores, hours))

	return &engine.Estimate{
		Name:           d.Address,
		CostComponents: costComponents,
	}
}

func synapseSparkPoolNodeSize(sizeName string) *decimal.Decimal {
	switch sizeName {
	case "Small":
		return decimalPtr(decimal.NewFromInt(4))
	case "Medium":
		return decimalPtr(decimal.NewFromInt(8))
	case "Large":
		return decimalPtr(decimal.NewFromInt(16))
	case "XLarge":
		return decimalPtr(decimal.NewFromInt(32))
	case "XXLarge":
		return decimalPtr(decimal.NewFromInt(64))
	default:
		return nil
	}
}

func synapseSparkPoolCostComponent(region, name, start string, instances, vCores, hours *decimal.Decimal) *engine.LineItem {

	var hourlyQuantity *decimal.Decimal
	if instances != nil && vCores != nil && hours != nil {
		hourlyQuantity = decimalPtr(vCores.Mul(*instances).Mul(*hours))
	}

	return &engine.LineItem{
		Name:            name,
		Unit:            "vCore-hours",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: hourlyQuantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Azure Synapse Analytics"),
			ProductFamily: strPtr("Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Azure Synapse Analytics Serverless Apache Spark Pool - Memory Optimized")},
				{Key: "skuName", Value: strPtr("vCore")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr(start),
		},
	}
}
