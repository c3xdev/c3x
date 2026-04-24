package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

func GetAzureRMCosmosdbCassandraKeyspaceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_cosmosdb_cassandra_keyspace",
		RFunc: NewAzureRMCosmosdb,
		ReferenceAttributes: []string{
			"account_name",
			"resource_group_name",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			if len(d.References("account_name")) > 0 {
				account := d.References("account_name")[0]
				return lookupRegion(account, []string{"account_name", "resource_group_name"})
			}

			return ""
		},
	}
}

type modelType int

const (
	Provisioned modelType = iota
	Autoscale
	Serverless
)

func NewAzureRMCosmosdb(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	if len(d.References("account_name")) > 0 {
		account := d.References("account_name")[0]
		return &engine.Estimate{
			Name:           d.Address,
			CostComponents: cosmosDBCostComponents(d, u, account),
		}
	}
	logging.Logger.Warn().Msgf("Skipping resource %s as its 'account_name' property could not be found.", d.Address)
	return nil
}

func cosmosDBCostComponents(d *engine.ResourceSpec, u *engine.ConsumptionProfile, account *engine.ResourceSpec) []*engine.LineItem {
	// Find the region in from the passed-in account
	region := d.Region
	geoLocations := account.Get("geo_location").Array()

	// The geo_location attribute is a required attribute however it can be an empty list because of
	// expressions evaluating as nil, e.g. using a data block. If the geoLocations variable is empty
	// we set it as a sane default which is using the location from the parent region.
	if len(geoLocations) == 0 {
		logging.Logger.Debug().Str(
			"resource", d.Address,
		).Msgf("empty set of geo_location attributes provided using fallback region %s", region)

		geoLocations = []gjson.Result{
			gjson.Parse(fmt.Sprintf(`{
    "location": %q,
    "failover_priority": 1
  }`, region)),
		}
	}

	costComponents := []*engine.LineItem{}

	model := Provisioned
	skuName := "RUs"
	if account.Get("enable_multiple_write_locations").Type != gjson.Null {
		if account.Get("enable_multiple_write_locations").Bool() {
			skuName = "mRUs"
		}
	}

	var throughputs *decimal.Decimal
	if d.Get("throughput").Type != gjson.Null {
		throughputs = decimalPtr(decimal.NewFromInt(d.Get("throughput").Int()))
	} else if d.Get("autoscale_settings.0.max_throughput").Type != gjson.Null {
		throughputs = decimalPtr(decimal.NewFromInt(d.Get("autoscale_settings.0.max_throughput").Int()))
		model = Autoscale
	} else {
		model = Serverless
		availabilityZone := geoLocations[0].Get("zone_zone_redundant").Bool()
		location := geoLocations[0].Get("location").String()
		costComponents = append(costComponents, serverlessCosmosCostComponent(location, availabilityZone, u))
	}

	if model == Provisioned || model == Autoscale {
		costComponents = provisionedCosmosCostComponents(
			model,
			throughputs,
			geoLocations,
			skuName,
			u)
	}

	costComponents = append(costComponents, storageCosmosCostComponents(account, u, geoLocations, skuName)...)

	backupType := "Pereodic"
	if account.Get("backup.0.type").Type != gjson.Null {
		backupType = account.Get("backup.0.type").String()
	}
	costComponents = append(costComponents, backupStorageCosmosCostComponents(account, u, geoLocations, backupType, region)...)

	return costComponents
}

func provisionedCosmosCostComponents(model modelType, throughputs *decimal.Decimal, zones []gjson.Result, skuName string, u *engine.ConsumptionProfile) []*engine.LineItem {
	costComponents := []*engine.LineItem{}

	var meterName string
	if strings.ToLower(skuName) == "rus" {
		meterName = "100 RU/s"
	} else {
		meterName = "100 Multi-master RU/s"
	}

	name := "Provisioned throughput"
	if model == Autoscale {
		name = fmt.Sprintf("%s (autoscale", name)

		if u != nil && u.Get("max_request_units_utilization_percentage").Exists() {
			throughputs = decimalPtr(throughputs.Mul(decimal.NewFromFloat(u.Get("max_request_units_utilization_percentage").Float() / 100)))
		} else {
			throughputs = nil
		}
	} else {
		name = fmt.Sprintf("%s (provisioned", name)
	}

	if throughputs != nil {
		throughputs = decimalPtr(throughputs.Div(decimal.NewFromInt(100)))
	}

	for _, g := range zones {
		quantity := throughputs

		if model == Autoscale {
			if strings.ToLower(skuName) == "rus" && quantity != nil {
				quantity = decimalPtr(quantity.Mul(decimal.NewFromFloat(1.5)))
			}
		} else {
			if strings.ToLower(skuName) == "rus" && quantity != nil {
				if g.Get("zone_redundant").Type != gjson.Null {
					if g.Get("zone_redundant").Bool() {
						quantity = decimalPtr(quantity.Mul(decimal.NewFromFloat(1.25)))
					}
				}
			}
		}

		location := g.Get("location").String()
		costComponents = append(costComponents, &engine.LineItem{
			Name:           fmt.Sprintf("%s, %s)", name, locationNameMapping(location)),
			Unit:           "RU/s x 100",
			UnitMultiplier: engine.HourToMonthUnitMultiplier,
			HourlyQuantity: quantity,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(location),
				Service:       strPtr("Azure Cosmos DB"),
				ProductFamily: strPtr("Databases"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "meterName", Value: strPtr(meterName)},
					{Key: "skuName", Value: strPtr(skuName)},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		})
	}

	return costComponents
}

func serverlessCosmosCostComponent(location string, availabilityZone bool, u *engine.ConsumptionProfile) *engine.LineItem {
	var requestUnits *decimal.Decimal
	if u != nil && u.Get("monthly_serverless_request_units").Exists() {
		requestUnits = decimalPtr(decimal.NewFromInt(u.Get("monthly_serverless_request_units").Int()))
		requestUnits = decimalPtr(requestUnits.Div(decimal.NewFromInt(1000000)))
	}

	if availabilityZone {
		requestUnits = decimalPtr(requestUnits.Mul(decimal.NewFromFloat(1.25)))
	}

	return &engine.LineItem{
		Name:            "Provisioned throughput (serverless)",
		Unit:            "1M RU",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: requestUnits,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(location),
			Service:       strPtr("Azure Cosmos DB"),
			ProductFamily: strPtr("Databases"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Azure Cosmos DB serverless")},
				{Key: "skuName", Value: strPtr("RUs")},
				{Key: "meterName", Value: strPtr("1M RUs")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func storageCosmosCostComponents(account *engine.ResourceSpec, u *engine.ConsumptionProfile, zones []gjson.Result, skuName string) []*engine.LineItem {
	costComponents := []*engine.LineItem{}
	var storageGB *decimal.Decimal
	if u != nil && u.Get("storage_gb").Exists() {
		storageGB = decimalPtr(decimal.NewFromInt(u.Get("storage_gb").Int()))
	}

	for _, g := range zones {
		location := g.Get("location").String()
		l := locationNameMapping(location)
		costComponents = append(costComponents, storageCosmosCostComponent(
			fmt.Sprintf("Transactional storage (%s)", l),
			location,
			skuName,
			"Azure Cosmos DB",
			storageGB))

		if account.Get("analytical_storage_enabled").Type != gjson.Null {
			if account.Get("analytical_storage_enabled").Bool() {
				costComponents = append(costComponents, storageCosmosCostComponent(
					fmt.Sprintf("Analytical storage (%s)", l),
					location,
					"Standard",
					"Azure Cosmos DB Analytics Storage",
					storageGB))

				var writeOperations, readOperations *decimal.Decimal
				if u != nil && u.Get("monthly_analytical_storage_write_operations").Exists() {
					writeOperations = decimalPtr(decimal.NewFromInt(u.Get("monthly_analytical_storage_write_operations").Int()))
					writeOperations = decimalPtr(writeOperations.Div(decimal.NewFromInt(10000)))
				}
				if u != nil && u.Get("monthly_analytical_storage_read_operations").Exists() {
					readOperations = decimalPtr(decimal.NewFromInt(u.Get("monthly_analytical_storage_read_operations").Int()))
					readOperations = decimalPtr(readOperations.Div(decimal.NewFromInt(10000)))
				}
				costComponents = append(costComponents, operationsCosmosCostComponent(
					fmt.Sprintf("Analytical write operations (%s)", l),
					location,
					"Write Operations",
					writeOperations,
				))

				costComponents = append(costComponents, operationsCosmosCostComponent(
					fmt.Sprintf("Analytical read operations (%s)", l),
					location,
					"Read Operations",
					readOperations,
				))
			}
		}
	}

	return costComponents
}

func backupStorageCosmosCostComponents(account *engine.ResourceSpec, u *engine.ConsumptionProfile, zones []gjson.Result, backupType, region string) []*engine.LineItem {
	costComponents := []*engine.LineItem{}
	var backupStorageGB *decimal.Decimal
	if u != nil && u.Get("storage_gb").Exists() {
		backupStorageGB = decimalPtr(decimal.NewFromInt(u.Get("storage_gb").Int()))
	}

	var name, meterName, skuName, productName string
	numberOfCopies := decimalPtr(decimal.NewFromInt(1))
	if strings.ToLower(backupType) == "periodic" {
		name = "Periodic backup"
		meterName = "Data Stored"
		skuName = "Standard"
		productName = "Azure Cosmos DB Snapshot"

		if backupStorageGB != nil {
			intervalHrs := 4.0
			retentionHrs := 8.0

			if account.Get("backup.0.interval_in_minutes").Type != gjson.Null {
				intervalHrs = account.Get("backup.0.interval_in_minutes").Float() / 60
			}
			if account.Get("backup.0.retention_in_hours").Type != gjson.Null {
				retentionHrs = account.Get("backup.0.retention_in_hours").Float()
			}

			if retentionHrs > intervalHrs {
				numberOfCopies = decimalPtr(decimal.NewFromFloat((retentionHrs / intervalHrs)).Floor().Sub(decimal.NewFromInt(2)))
			}
			backupStorageGB = decimalPtr(backupStorageGB.Mul(*numberOfCopies))
		}
	} else {
		name = "Continuous backup"
		meterName = "Continuous Backup"
		skuName = "Backup"
		productName = "Azure Cosmos DB - PITR"
	}

	for _, g := range zones {
		if backupStorageGB != nil {
			if backupStorageGB.Equal(decimal.Zero) {
				break
			}
		}
		location := g.Get("location").String()
		costComponents = append(costComponents, backupCosmosCostComponent(
			fmt.Sprintf("%s (%s)", name, locationNameMapping(location)),
			location,
			skuName,
			productName,
			meterName,
			backupStorageGB,
		))
	}

	var pitr *decimal.Decimal
	if u != nil && u.Get("monthly_restored_data_gb").Exists() {
		pitr = decimalPtr(decimal.NewFromInt(u.Get("monthly_restored_data_gb").Int()))
	}
	meterName = "Data Restore"
	skuName = "Backup"
	productName = "Azure Cosmos DB - PITR"

	costComponents = append(costComponents, backupCosmosCostComponent(
		"Restored data",
		region,
		skuName,
		productName,
		meterName,
		pitr,
	))

	return costComponents
}

func storageCosmosCostComponent(name, location, skuName, productName string, quantities *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantities,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(location),
			Service:       strPtr("Azure Cosmos DB"),
			ProductFamily: strPtr("Databases"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr("Data Stored$")},
				{Key: "skuName", Value: strPtr(skuName)},
				{Key: "productName", Value: strPtr(productName)},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func backupCosmosCostComponent(name, location, skuName, productName, meterName string, quantities *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:                 name,
		Unit:                 "GB",
		UnitMultiplier:       decimal.NewFromInt(1),
		MonthlyQuantity:      quantities,
		IgnoreIfMissingPrice: true,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(location),
			Service:       strPtr("Azure Cosmos DB"),
			ProductFamily: strPtr("Databases"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr(meterName)},
				{Key: "skuName", Value: strPtr(skuName)},
				{Key: "productName", Value: strPtr(productName)},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func operationsCosmosCostComponent(name, location, meterName string, quantities *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "10K operations",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantities,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(location),
			Service:       strPtr("Azure Cosmos DB"),
			ProductFamily: strPtr("Databases"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("%s$", meterName))},
				{Key: "skuName", Value: strPtr("Standard")},
				{Key: "productName", Value: strPtr("Azure Cosmos DB Analytics Storage")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
