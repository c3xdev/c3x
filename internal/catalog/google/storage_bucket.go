package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type StorageBucket struct {
	Address                     string
	Region                      string
	Location                    string
	StorageClass                string
	StorageGB                   *float64                         `c3x_usage:"storage_gb"`
	MonthlyClassAOperations     *int64                           `c3x_usage:"monthly_class_a_operations"`
	MonthlyClassBOperations     *int64                           `c3x_usage:"monthly_class_b_operations"`
	MonthlyDataRetrievalGB      *float64                         `c3x_usage:"monthly_data_retrieval_gb"`
	MonthlyEgressDataTransferGB *StorageBucketNetworkEgressUsage `c3x_usage:"monthly_egress_data_transfer_gb"`
}

func (r *StorageBucket) CoreType() string {
	return "StorageBucket"
}

func (r *StorageBucket) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_class_a_operations", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_class_b_operations", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_data_retrieval_gb", ValueType: engine.Float64, DefaultValue: 0},
		{
			Key:          "monthly_egress_data_transfer_gb",
			ValueType:    engine.SubResourceUsage,
			DefaultValue: &usage.ResourceUsage{Name: "monthly_egress_data_transfer_gb", Items: StorageBucketNetworkEgressUsageSchema},
		},
	}
}

func (r *StorageBucket) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *StorageBucket) BuildResource() *engine.Estimate {
	if r.MonthlyEgressDataTransferGB == nil {
		r.MonthlyEgressDataTransferGB = &StorageBucketNetworkEgressUsage{}
	}
	region := r.Region
	components := []*engine.LineItem{
		dataStorageCostComponent(r.Location, r.StorageClass, r.StorageGB),
	}
	data := dataRetrievalCostComponent(r)
	if data != nil {
		components = append(components, data)
	}
	components = append(components, operationsCostComponents(r.StorageClass, r.MonthlyClassAOperations, r.MonthlyClassBOperations)...)

	r.MonthlyEgressDataTransferGB.Region = region
	r.MonthlyEgressDataTransferGB.Address = "Network egress"
	r.MonthlyEgressDataTransferGB.PrefixName = "Data transfer"
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: components,
		SubResources: []*engine.Estimate{
			r.MonthlyEgressDataTransferGB.BuildResource(),
		}, UsageSchema: r.UsageSchema(),
	}
}

func getDSRegionResourceGroup(location, storageClass string) (string, string) {

	region := strings.ToLower(location)

	var resourceGroup string
	switch strings.ToLower(storageClass) {
	case "nearline":
		resourceGroup = "NearlineStorage"
	case "coldline":
		resourceGroup = "ColdlineStorage"
	case "archive":
		resourceGroup = "ArchiveStorage"
	default:
		resourceGroup = "RegionalStorage"
	}

	if strings.ToLower(resourceGroup) == "regionalstorage" {
		switch region {

		case "asia", "eu", "us":
			resourceGroup = "MultiRegionalStorage"

		case "asia1", "eur4", "nam4":

			resourceGroup = "MultiRegionalStorage"
		}
	}

	if region == "eu" && strings.ToLower(resourceGroup) == "multiregionalstorage" {
		region = "europe"
	}

	return region, resourceGroup
}

func dataStorageCostComponent(location, storageClass string, storageGB *float64) *engine.LineItem {
	if location == "" {
		location = "US"
	}

	var quantity *decimal.Decimal
	if storageGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*storageGB))
	}
	if storageClass == "" {
		storageClass = "STANDARD"
	}

	region, resourceGroup := getDSRegionResourceGroup(location, storageClass)
	return &engine.LineItem{
		Name:            fmt.Sprintf("Storage (%s)", strings.ToLower(storageClass)),
		Unit:            "GiB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("gcp"),
			Region:     strPtr(region),
			Service:    strPtr("Cloud Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "resourceGroup", Value: strPtr(resourceGroup)},
				{Key: "description", ValueRegex: strPtr("/^(?!.*?\\(Early Delete\\))/")},
			},
		},
		PriceFilter: &engine.RateSelector{
			EndUsageAmount: strPtr(""),
		},
		UsageBased: true,
	}
}

func operationsCostComponents(storageClass string, monthlyClassAOperations, monthlyClassBOperations *int64) []*engine.LineItem {
	var classAQuantity *decimal.Decimal
	if monthlyClassAOperations != nil {
		classAQuantity = decimalPtr(decimal.NewFromInt(*monthlyClassAOperations))
	}
	var classBQuantity *decimal.Decimal
	if monthlyClassBOperations != nil {
		classBQuantity = decimalPtr(decimal.NewFromInt(*monthlyClassBOperations))
	}
	if storageClass == "" {
		storageClass = "STANDARD"
	}

	storageClassResourceGroupMap := map[string]string{
		"STANDARD":       "RegionalOps",
		"REGIONAL":       "RegionalOps",
		"MULTI_REGIONAL": "MultiRegionalOps",
		"NEARLINE":       "NearlineOps",
		"COLDLINE":       "ColdlineOps",
		"ARCHIVE":        "ArchiveOps",
	}

	resourceGroup := storageClassResourceGroupMap[storageClass]

	var descriptionRegex string
	switch resourceGroup {
	case "RegionalOps":
		descriptionRegex = "^(?!Multi-Region|Dual-Region)(?:(?!Tagging).)*"
	case "MultiRegionalOps":
		descriptionRegex = "^(?:(?!Tagging).)*"
	default:
		descriptionRegex = "^(?!Regional|Multi-Region|Dual-Region|Region)(?:(?!Tagging).)*"
	}

	return []*engine.LineItem{
		{
			Name:            "Object adds, bucket/object list (class A)",
			Unit:            "10k operations",
			UnitMultiplier:  decimal.NewFromInt(10000),
			MonthlyQuantity: classAQuantity,
			ProductFilter: &engine.ProductSelector{
				VendorName: strPtr("gcp"),
				Service:    strPtr("Cloud Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "resourceGroup", Value: strPtr(resourceGroup)},
					{Key: "description", ValueRegex: regexPtr(fmt.Sprintf("%sClass A", descriptionRegex))},
				},
			},
			PriceFilter: &engine.RateSelector{
				EndUsageAmount: strPtr(""),
			},
			UsageBased: true,
		},
		{
			Name:            "Object gets, retrieve bucket/object metadata (class B)",
			Unit:            "10k operations",
			UnitMultiplier:  decimal.NewFromInt(10000),
			MonthlyQuantity: classBQuantity,
			ProductFilter: &engine.ProductSelector{
				VendorName: strPtr("gcp"),
				Service:    strPtr("Cloud Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "resourceGroup", Value: strPtr(resourceGroup)},
					{Key: "description", ValueRegex: regexPtr(fmt.Sprintf("%sClass B", descriptionRegex))},
				},
			},
			PriceFilter: &engine.RateSelector{
				EndUsageAmount: strPtr(""),
			},
			UsageBased: true,
		},
	}
}

func dataRetrievalCostComponent(r *StorageBucket) *engine.LineItem {
	var quantity *decimal.Decimal
	if r.MonthlyDataRetrievalGB != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataRetrievalGB))
	}

	storageClass := "STANDARD"
	if r.StorageClass != "" {
		storageClass = r.StorageClass
	}

	storageClassResourceGroupMap := map[string]string{
		"NEARLINE": "NearlineOps",
		"COLDLINE": "ColdlineOps",
		"ARCHIVE":  "ArchiveOps",
	}
	resourceGroup := storageClassResourceGroupMap[storageClass]

	if resourceGroup == "" {
		return nil
	}

	return &engine.LineItem{
		Name:            "Data retrieval",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("gcp"),
			Service:    strPtr("Cloud Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "resourceGroup", Value: strPtr(resourceGroup)},
				{Key: "description", ValueRegex: strPtr("/Retrieval/")},
			},
		},
		UsageBased: true,
	}
}
