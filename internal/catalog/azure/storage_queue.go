package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
)

// StorageQueue struct represents Azure Queue Storage.
//
// Resource information: https://azure.microsoft.com/en-gb/pricing/details/storage/queues/
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/storage/queues/#pricing
type StorageQueue struct {
	Address                string
	Region                 string
	AccountKind            string
	AccountReplicationType string

	MonthlyStorageGB                    *float64 `c3x_usage:"monthly_storage_gb"`
	MonthlyClass1Operations             *int64   `c3x_usage:"monthly_class_1_operations"`
	MonthlyClass2Operations             *int64   `c3x_usage:"monthly_class_2_operations"`
	MonthlyGeoReplicationDataTransferGB *float64 `c3x_usage:"monthly_geo_replication_data_transfer_gb"`
}

// CoreType returns the name of this resource type
func (r *StorageQueue) CoreType() string {
	return "StorageQueue"
}

// UsageSchema defines a list which represents the usage schema of StorageQueue.
func (r *StorageQueue) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_storage_gb", DefaultValue: 0.0, ValueType: engine.Float64},
		{Key: "monthly_class_1_operations", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_class_2_operations", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_geo_replication_data_transfer_gb", DefaultValue: 0.0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the StorageQueue.
// It uses the `c3x_usage` struct tags to populate data into the StorageQueue.
func (r *StorageQueue) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid StorageQueue struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *StorageQueue) BuildResource() *engine.Estimate {
	if !r.isAccountKindSupported() {
		logging.Logger.Warn().Msgf("Skipping resource %s. Storage Queues don't support %s accounts", r.Address, r.AccountKind)
		return nil
	}

	if !r.isReplicationTypeSupported() {
		logging.Logger.Warn().Msgf("Skipping resource %s. Storage Queues don't support %s redundancy", r.Address, r.AccountReplicationType)
		return nil
	}

	costComponents := []*engine.LineItem{
		r.dataStorageCostComponent(),
	}
	costComponents = append(costComponents, r.operationsCostComponents()...)
	costComponents = append(costComponents, r.geoReplicationDataTransferCostComponents()...)

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *StorageQueue) isAccountKindSupported() bool {
	return r.isStorageV1() || r.isStorageV2()
}

func (r *StorageQueue) isReplicationTypeSupported() bool {
	var validReplicationTypes []string

	switch {
	case r.isStorageV1():
		validReplicationTypes = []string{"LRS", "GRS", "RA-GRS"}
	case r.isStorageV2():
		validReplicationTypes = []string{"LRS", "ZRS", "GRS", "RA-GRS", "GZRS", "RA-GZRS"}
	}

	if validReplicationTypes != nil {
		return contains(validReplicationTypes, strings.ToUpper(r.AccountReplicationType))
	}

	return true
}

func (r *StorageQueue) isStorageV1() bool {
	return strings.EqualFold(r.AccountKind, "storage")
}

func (r *StorageQueue) isStorageV2() bool {
	return strings.EqualFold(r.AccountKind, "storagev2")
}

func (r *StorageQueue) productName() string {
	if r.isStorageV1() {
		return "Queues"
	}

	return "Queues v2"
}

func (r *StorageQueue) dataStorageCostComponent() *engine.LineItem {
	var qty *decimal.Decimal
	if r.MonthlyStorageGB != nil {
		qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyStorageGB))
	}

	return &engine.LineItem{
		Name:            "Capacity",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: qty,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Storage"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr(r.productName())},
				{Key: "skuName", Value: strPtr(fmt.Sprintf("Standard %s", strings.ToUpper(r.AccountReplicationType)))},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Data Stored", strings.ToUpper(r.AccountReplicationType)))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}
}

func (r *StorageQueue) operationsCostComponents() []*engine.LineItem {
	costComponents := []*engine.LineItem{}

	if !contains([]string{"GZRS", "RA-GZRS"}, strings.ToUpper(r.AccountReplicationType)) {
		var class1Qty *decimal.Decimal
		if r.MonthlyClass1Operations != nil {
			class1Qty = decimalPtr(decimal.NewFromInt(*r.MonthlyClass1Operations).Div(decimal.NewFromInt(10000)))
		}

		costComponents = append(costComponents, &engine.LineItem{
			Name:            "Class 1 operations",
			Unit:            "10k operations",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: class1Qty,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Storage"),
				ProductFamily: strPtr("Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "productName", Value: strPtr(r.productName())},
					{Key: "skuName", Value: strPtr(fmt.Sprintf("Standard %s", strings.ToUpper(r.AccountReplicationType)))},
					{Key: "meterName", ValueRegex: regexPtr("Class 1 Operations$")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption:   strPtr("Consumption"),
				StartUsageAmount: strPtr("0"),
			},
			UsageBased: true,
		})
	}

	var class2Qty *decimal.Decimal
	if r.MonthlyClass1Operations != nil {
		class2Qty = decimalPtr(decimal.NewFromInt(*r.MonthlyClass2Operations).Div(decimal.NewFromInt(10000)))
	}

	costComponents = append(costComponents, &engine.LineItem{
		Name:            "Class 2 operations",
		Unit:            "10k operations",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: class2Qty,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Storage"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr(r.productName())},
				{Key: "skuName", Value: strPtr(fmt.Sprintf("Standard %s", strings.ToUpper(r.AccountReplicationType)))},
				{Key: "meterName", ValueRegex: regexPtr("Class 2 Operations$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	})

	return costComponents
}

func (r *StorageQueue) geoReplicationDataTransferCostComponents() []*engine.LineItem {
	if contains([]string{"LRS", "ZRS"}, strings.ToUpper(r.AccountReplicationType)) {
		return []*engine.LineItem{}
	}

	var qty *decimal.Decimal
	if r.MonthlyGeoReplicationDataTransferGB != nil {
		qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyGeoReplicationDataTransferGB))
	}

	return []*engine.LineItem{
		{
			Name:            "Geo-replication data transfer",
			Unit:            "GB",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: qty,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Storage"),
				ProductFamily: strPtr("Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "productName", Value: strPtr("Storage - Bandwidth")},
					{Key: "skuName", Value: strPtr("Geo-Replication v2")},
					{Key: "meterName", Value: strPtr("Geo-Replication v2 Data Transfer")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption:   strPtr("Consumption"),
				StartUsageAmount: strPtr("0"),
			},
			UsageBased: true,
		},
	}
}
