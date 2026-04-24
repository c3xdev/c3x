package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
)

func GetAzureRMRedisCacheRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_redis_cache",
		RFunc: NewAzureRMRedisCache,
	}
}

func NewAzureRMRedisCache(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	region := d.Region

	skuName := d.Get("sku_name").String()
	family := d.Get("family").String()
	capacity := d.Get("capacity").String()

	sku := family + capacity
	productName := fmt.Sprintf("Azure Redis Cache %s", skuName)

	nodesPerShard := map[string]int64{
		"basic":    1,
		"standard": 2,
		"premium":  2,
	}[strings.ToLower(skuName)]

	shards := int64(1)

	if strings.EqualFold(skuName, "premium") {
		// If shards is set to 0 then we still want to calculate the cost for 1 shard
		// since the cache will be in HA-mode not cluster mode.
		if d.Get("shard_count").Int() > 0 {
			shards = d.Get("shard_count").Int()
		}

		if d.Get("replicas_per_primary").Type != gjson.Null {
			nodesPerShard = 1 + d.Get("replicas_per_primary").Int()
		} else if d.Get("replicas_per_master").Type != gjson.Null {
			nodesPerShard = 1 + d.Get("replicas_per_master").Int()
		}
	}

	nodes := shards * nodesPerShard

	qty := decimal.NewFromInt(nodes)
	mul := engine.HourToMonthUnitMultiplier
	attributes := []*engine.AttributeMatch{
		{Key: "productName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", productName))},
		{Key: "skuName", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", sku))},
	}

	if strings.EqualFold(skuName, "premium") {
		attributes = append(attributes,
			&engine.AttributeMatch{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("/^%s Cache Instance$/i", sku))},
		)
	} else {
		attributes = append(attributes,
			&engine.AttributeMatch{Key: "meterName", ValueRegex: strPtr(fmt.Sprintf("/^%s Cache$/i", sku))},
		)
	}

	if strings.EqualFold(skuName, "standard") {
		qty = qty.Div(decimal.NewFromInt(2))
		mul = mul.Div(decimal.NewFromInt(2))
	}

	return &engine.Estimate{
		Name: d.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:           fmt.Sprintf("Cache usage (%s_%s%s)", skuName, family, capacity),
				Unit:           "nodes",
				UnitMultiplier: mul,
				HourlyQuantity: decimalPtr(qty),
				ProductFilter: &engine.ProductSelector{
					VendorName:       strPtr("azure"),
					Region:           strPtr(region),
					Service:          strPtr("Redis Cache"),
					ProductFamily:    strPtr("Databases"),
					AttributeFilters: attributes,
				},
				PriceFilter: &engine.RateSelector{
					PurchaseOption: strPtr("Consumption"),
				},
			},
		},
	}

}
