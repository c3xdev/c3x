package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type BigQueryTable struct {
	Address                   string
	Region                    string
	MonthlyStreamingInsertsMB *float64 `c3x_usage:"monthly_streaming_inserts_mb"`
	MonthlyStorageWriteAPIGB  *float64 `c3x_usage:"monthly_storage_write_api_gb"`
	MonthlyStorageReadAPITB   *float64 `c3x_usage:"monthly_storage_read_api_tb"`
	MonthlyActiveStorageGB    *float64 `c3x_usage:"monthly_active_storage_gb"`
	MonthlyLongTermStorageGB  *float64 `c3x_usage:"monthly_long_term_storage_gb"`
}

func (r *BigQueryTable) CoreType() string {
	return "BigQueryTable"
}

func (r *BigQueryTable) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_streaming_inserts_mb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_storage_write_api_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_storage_read_api_tb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_active_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_long_term_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *BigQueryTable) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *BigQueryTable) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.activeStorageCostComponent(),
		r.longTermStorageCostComponent(),
		r.streamingInsertsCostComponent(),
	}

	storageWriteAPICostComponent := r.storageWriteAPICostComponent()
	if storageWriteAPICostComponent != nil {
		costComponents = append(costComponents, storageWriteAPICostComponent)
	}

	storageReadAPICostComponent := r.storageReadAPICostComponent()
	if storageReadAPICostComponent != nil {
		costComponents = append(costComponents, storageReadAPICostComponent)
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *BigQueryTable) activeStorageCostComponent() *engine.LineItem {
	var activeStorageGB *decimal.Decimal
	if r.MonthlyActiveStorageGB != nil {
		activeStorageGB = decimalPtr(decimal.NewFromFloat(*r.MonthlyActiveStorageGB))
	}

	return &engine.LineItem{
		Name:            "Active storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: activeStorageGB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("BigQuery"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", Value: strPtr(fmt.Sprintf("Active Logical Storage (%s)", r.Region))},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("10"),
		},
		UsageBased: true,
	}
}

func (r *BigQueryTable) longTermStorageCostComponent() *engine.LineItem {
	var longTermStorageGB *decimal.Decimal
	if r.MonthlyLongTermStorageGB != nil {
		longTermStorageGB = decimalPtr(decimal.NewFromFloat(*r.MonthlyLongTermStorageGB))
	}

	return &engine.LineItem{
		Name:            "Long-term storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: longTermStorageGB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("BigQuery"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", Value: strPtr(fmt.Sprintf("Long Term Logical Storage (%s)", r.Region))},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("10"),
		},
		UsageBased: true,
	}
}

func (r *BigQueryTable) streamingInsertsCostComponent() *engine.LineItem {
	var streamingInsertsMB *decimal.Decimal
	if r.MonthlyStreamingInsertsMB != nil {
		streamingInsertsMB = decimalPtr(decimal.NewFromFloat(*r.MonthlyStreamingInsertsMB))
	}

	return &engine.LineItem{
		Name:            "Streaming inserts",
		Unit:            "MB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: streamingInsertsMB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("BigQuery"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", Value: strPtr(fmt.Sprintf("Streaming Insert (%s)", r.Region))},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}
}

func (r *BigQueryTable) storageWriteAPICostComponent() *engine.LineItem {
	var storageWriteAPIGB *decimal.Decimal
	if r.MonthlyStorageWriteAPIGB != nil {
		storageWriteAPIGB = decimalPtr(decimal.NewFromFloat(*r.MonthlyStorageWriteAPIGB))
	}

	region := r.mapRegion()
	if region == "" {
		return nil
	}

	return &engine.LineItem{
		Name:            "Storage write API",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: storageWriteAPIGB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(region),
			Service:       strPtr("BigQuery Storage API"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", Value: strPtr(fmt.Sprintf("BigQuery Storage API - Write (%s)", region))},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("2048"),
		},
		UsageBased: true,
	}
}

func (r *BigQueryTable) storageReadAPICostComponent() *engine.LineItem {
	var storageReadPITB *decimal.Decimal
	if r.MonthlyStorageReadAPITB != nil {
		storageReadPITB = decimalPtr(decimal.NewFromFloat(*r.MonthlyStorageReadAPITB))
	}

	region := r.mapRegion()
	if region == "" {
		return nil
	}

	return &engine.LineItem{
		Name:            "Storage read API",
		Unit:            "TB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: storageReadPITB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(region),
			Service:       strPtr("BigQuery Storage API"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", Value: strPtr("BigQuery Storage API - Read")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}
}

func (r *BigQueryTable) mapRegion() string {
	if strings.HasPrefix(strings.ToLower(r.Region), "us") {
		return "us"
	}
	if strings.HasPrefix(strings.ToLower(r.Region), "europe") {
		return "europe"
	}

	return ""
}
