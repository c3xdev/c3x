package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type ContainerRegistry struct {
	Address                 string
	GeoReplicationLocations int
	Region                  string
	SKU                     string
	StorageGB               *float64 `c3x_usage:"storage_gb"`
	MonthlyBuildVCPUHrs     *float64 `c3x_usage:"monthly_build_vcpu_hrs"`
}

func (r *ContainerRegistry) CoreType() string {
	return "ContainerRegistry"
}

func (r *ContainerRegistry) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_build_vcpu_hrs", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *ContainerRegistry) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ContainerRegistry) BuildResource() *engine.Estimate {

	var locationsCount int
	var storageGB, monthlyBuildVCPU *decimal.Decimal
	var overStorage decimal.Decimal

	sku := "Classic"
	includedStorage := decimal.NewFromFloat(10)

	if r.SKU != "" {
		sku = r.SKU
	}

	switch sku {
	case "Basic":
		includedStorage = decimal.NewFromFloat(10)
	case "Standard":
		includedStorage = decimal.NewFromFloat(100)
	case "Premium":
		includedStorage = decimal.NewFromFloat(500)
	}

	locationsCount = r.GeoReplicationLocations

	costComponents := make([]*engine.LineItem, 0)

	if locationsCount > 0 {
		suffix := fmt.Sprintf("%d locations", locationsCount)
		if locationsCount == 1 {
			suffix = fmt.Sprintf("%d location", locationsCount)
		}
		costComponents = append(costComponents, r.containerRegistryGeolocationCostComponent(fmt.Sprintf("Geo replication (%s)", suffix), sku))
	}

	costComponents = append(costComponents, r.containerRegistryCostComponent(fmt.Sprintf("Registry usage (%s)", sku), sku))

	if r.StorageGB != nil {
		storageGB = decimalPtr(decimal.NewFromFloat(*r.StorageGB))
		if storageGB.GreaterThan(includedStorage) {
			overStorage = storageGB.Sub(includedStorage)
			storageGB = &overStorage
			costComponents = append(costComponents, r.containerRegistryStorageCostComponent(fmt.Sprintf("Storage (over %sGB)", includedStorage), sku, storageGB))
		}
	} else {
		costComponents = append(costComponents, r.containerRegistryStorageCostComponent(fmt.Sprintf("Storage (over %sGB)", includedStorage), sku, storageGB))
	}

	if r.MonthlyBuildVCPUHrs != nil {
		monthlyBuildVCPU = decimalPtr(decimal.NewFromFloat(*r.MonthlyBuildVCPUHrs * 3600))
	}

	costComponents = append(costComponents, r.containerRegistryCPUCostComponent("Build vCPU", sku, monthlyBuildVCPU))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *ContainerRegistry) containerRegistryCostComponent(name, sku string) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "days",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(30)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Container Registry"),
			ProductFamily: strPtr("Containers"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Container Registry")},
				{Key: "skuName", Value: strPtr(sku)},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Registry Unit", sku))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
func (r *ContainerRegistry) containerRegistryGeolocationCostComponent(name, sku string) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "days",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(30 * int64(r.GeoReplicationLocations))),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Container Registry"),
			ProductFamily: strPtr("Containers"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Container Registry")},
				{Key: "skuName", Value: strPtr(sku)},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Registry Unit", sku))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}
func (r *ContainerRegistry) containerRegistryStorageCostComponent(name, sku string, storage *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{

		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: storage,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Container Registry"),
			ProductFamily: strPtr("Containers"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Container Registry")},
				{Key: "skuName", Value: strPtr(sku)},
				{Key: "meterName", Value: strPtr("Data Stored")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
		UsageBased: true,
	}
}
func (r *ContainerRegistry) containerRegistryCPUCostComponent(name, sku string, monthlyBuildVCPU *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{

		Name:            name,
		Unit:            "seconds",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monthlyBuildVCPU,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Container Registry"),
			ProductFamily: strPtr("Containers"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr("Container Registry")},
				{Key: "skuName", Value: strPtr(sku)},
				{Key: "meterName", Value: strPtr("Task vCPU Duration")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("6000"),
		},
		UsageBased: true,
	}
}
