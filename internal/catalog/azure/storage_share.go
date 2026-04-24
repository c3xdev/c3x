package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// StorageShare struct represents an Azure Files Storage Shares
//
// Resource information: https://azure.microsoft.com/en-gb/pricing/details/storage/files/
// Pricing information: https://azure.microsoft.com/en-gb/pricing/details/storage/files/#pricing
type StorageShare struct {
	Address                string
	Region                 string
	AccountReplicationType string
	AccessTier             string
	Quota                  int64

	// "usage" args
	MonthlyStorageGB        *float64 `c3x_usage:"storage_gb"`
	MonthlyReadOperations   *int64   `c3x_usage:"monthly_read_operations"`
	MonthlyWriteOperations  *int64   `c3x_usage:"monthly_write_operations"`
	MonthlyListOperations   *int64   `c3x_usage:"monthly_list_operations"`
	MonthlyOtherOperations  *int64   `c3x_usage:"monthly_other_operations"`
	MonthlyDataRetrievalGB  *float64 `c3x_usage:"monthly_data_retrieval_gb"`
	SnapshotsStorageGB      *float64 `c3x_usage:"snapshots_storage_gb"`
	MetadataAtRestStorageGB *float64 `c3x_usage:"metadata_at_rest_storage_gb"`
}

// CoreType returns the name of this resource type
func (r *StorageShare) CoreType() string {
	return "StorageShare"
}

// UsageSchema defines a list which represents the usage schema of StorageShare.
func (r *StorageShare) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "storage_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "monthly_read_operations", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_write_operations", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_list_operations", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_other_operations", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_data_retrieval_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "snapshots_storage_gb", DefaultValue: 0, ValueType: engine.Float64},
		{Key: "metadata_at_rest_storage_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the StorageShare.
// It uses the `c3x_usage` struct tags to populate data into the StorageShare.
func (r *StorageShare) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid StorageShare struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *StorageShare) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.dataStorageCostComponent(),
	}

	costComponents = append(costComponents, r.snapshotCostComponents()...)
	costComponents = append(costComponents, r.metadataCostComponents()...)
	costComponents = append(costComponents, r.readOperationsCostComponents()...)
	costComponents = append(costComponents, r.writeOperationsCostComponents()...)
	costComponents = append(costComponents, r.listOperationsCostComponents()...)
	costComponents = append(costComponents, r.otherOperationsCostComponents()...)
	costComponents = append(costComponents, r.dataRetrievalCostComponents()...)

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *StorageShare) productName() string {
	if r.accessTier() == "Premium" {
		return "Premium Files"
	}

	return "Files v2"
}

func (r *StorageShare) accessTier() string {
	return map[string]string{
		"hot":                  "Hot",
		"cool":                 "Cool",
		"transactionoptimized": "Standard",
		"premium":              "Premium",
	}[strings.ToLower(r.AccessTier)]
}

func (r *StorageShare) dataStorageCostComponent() *engine.LineItem {
	var qty *decimal.Decimal

	if r.accessTier() == "Premium" {
		qty = decimalPtr(decimal.NewFromInt(r.Quota))
	}

	if r.MonthlyStorageGB != nil {
		qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyStorageGB))
	}

	skuName := fmt.Sprintf("%s %s", r.accessTier(), strings.ToUpper(r.AccountReplicationType))
	meterName := "Data Stored"
	if r.accessTier() == "Premium" {
		meterName = "Provisioned"
	}

	return &engine.LineItem{
		Name:            "Data at rest",
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
				{Key: "skuName", Value: strPtr(skuName)},
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("%s$", meterName))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}
}

func (r *StorageShare) snapshotCostComponents() []*engine.LineItem {
	var qty *decimal.Decimal
	if r.SnapshotsStorageGB != nil {
		qty = decimalPtr(decimal.NewFromFloat(*r.SnapshotsStorageGB))
	}

	skuName := fmt.Sprintf("%s %s", r.accessTier(), strings.ToUpper(r.AccountReplicationType))
	meterName := "Data Stored"
	if r.accessTier() == "Premium" {
		meterName = "Snapshots"
	}

	return []*engine.LineItem{{
		Name:            "Snapshots",
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
				{Key: "skuName", Value: strPtr(skuName)},
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("%s$", meterName))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}}
}

func (r *StorageShare) metadataCostComponents() []*engine.LineItem {
	if contains([]string{"Premium", "Standard"}, r.accessTier()) {
		return []*engine.LineItem{}
	}

	var qty *decimal.Decimal
	if r.MetadataAtRestStorageGB != nil {
		qty = decimalPtr(decimal.NewFromFloat(*r.MetadataAtRestStorageGB))
	}

	skuName := fmt.Sprintf("%s %s", r.accessTier(), strings.ToUpper(r.AccountReplicationType))

	return []*engine.LineItem{{
		Name:            "Metadata at rest",
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
				{Key: "skuName", Value: strPtr(skuName)},
				{Key: "meterName", ValueRegex: regexPtr("Metadata$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}}
}

func (r *StorageShare) readOperationsCostComponents() []*engine.LineItem {
	if r.accessTier() == "Premium" {
		return []*engine.LineItem{}
	}

	var qty *decimal.Decimal
	if r.MonthlyReadOperations != nil {
		qty = decimalPtr(decimal.NewFromInt(*r.MonthlyReadOperations).Div(decimal.NewFromInt(10000)))
	}

	skuName := fmt.Sprintf("%s %s", r.accessTier(), strings.ToUpper(r.AccountReplicationType))

	return []*engine.LineItem{{
		Name:            "Read operations",
		Unit:            "10k operations",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: qty,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Storage"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr(r.productName())},
				{Key: "skuName", Value: strPtr(skuName)},
				{Key: "meterName", ValueRegex: regexPtr("Read Operations$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}}
}

func (r *StorageShare) writeOperationsCostComponents() []*engine.LineItem {
	if r.accessTier() == "Premium" {
		return []*engine.LineItem{}
	}

	var qty *decimal.Decimal
	if r.MonthlyWriteOperations != nil {
		qty = decimalPtr(decimal.NewFromInt(*r.MonthlyWriteOperations).Div(decimal.NewFromInt(10000)))
	}

	skuName := fmt.Sprintf("%s %s", r.accessTier(), strings.ToUpper(r.AccountReplicationType))

	return []*engine.LineItem{{
		Name:            "Write operations",
		Unit:            "10k operations",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: qty,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Storage"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr(r.productName())},
				{Key: "skuName", Value: strPtr(skuName)},
				{Key: "meterName", ValueRegex: regexPtr("Write Operations$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}}
}

func (r *StorageShare) listOperationsCostComponents() []*engine.LineItem {
	if r.accessTier() == "Premium" {
		return []*engine.LineItem{}
	}

	var qty *decimal.Decimal
	if r.MonthlyListOperations != nil {
		qty = decimalPtr(decimal.NewFromInt(*r.MonthlyListOperations).Div(decimal.NewFromInt(10000)))
	}

	skuName := fmt.Sprintf("%s %s", r.accessTier(), strings.ToUpper(r.AccountReplicationType))

	return []*engine.LineItem{{
		Name:            "List operations",
		Unit:            "10k operations",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: qty,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Storage"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr(r.productName())},
				{Key: "skuName", Value: strPtr(skuName)},

				{Key: "meterName", ValueRegex: regexPtr("List Operations$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("Consumption"),
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}}
}

func (r *StorageShare) otherOperationsCostComponents() []*engine.LineItem {
	if r.accessTier() == "Premium" {
		return []*engine.LineItem{}
	}

	var qty *decimal.Decimal
	if r.MonthlyOtherOperations != nil {
		qty = decimalPtr(decimal.NewFromInt(*r.MonthlyOtherOperations).Div(decimal.NewFromInt(10000)))
	}

	skuName := fmt.Sprintf("%s %s", r.accessTier(), strings.ToUpper(r.AccountReplicationType))
	meterName := "Other Operations"
	if r.accessTier() == "Standard" {
		meterName = "Protocol Operations"
	}

	return []*engine.LineItem{
		{
			Name:            "Other operations",
			Unit:            "10k operations",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: qty,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(r.Region),
				Service:       strPtr("Storage"),
				ProductFamily: strPtr("Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "productName", Value: strPtr(r.productName())},
					{Key: "skuName", Value: strPtr(skuName)},
					{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("%s$", meterName))},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption:   strPtr("Consumption"),
				StartUsageAmount: strPtr("0"),
			},
			UsageBased: true,
		}}
}

func (r *StorageShare) dataRetrievalCostComponents() []*engine.LineItem {
	if contains([]string{"Premium", "Standard", "Hot"}, r.accessTier()) || strings.ToUpper(r.AccountReplicationType) == "GZRS" {
		return []*engine.LineItem{}
	}

	var qty *decimal.Decimal
	if r.MonthlyDataRetrievalGB != nil {
		qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataRetrievalGB))
	}

	skuName := fmt.Sprintf("%s %s", r.accessTier(), strings.ToUpper(r.AccountReplicationType))

	return []*engine.LineItem{
		{
			Name:            "Data retrieval",
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
					{Key: "skuName", Value: strPtr(skuName)},
					{Key: "meterName", ValueRegex: regexPtr("Data Retrieval$")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption:   strPtr("Consumption"),
				StartUsageAmount: strPtr("0"),
			},
			UsageBased: true,
		}}
}
