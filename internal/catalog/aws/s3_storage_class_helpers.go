package aws

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/engine"
)

func s3StorageCostComponent(name string, service string, region string, usageType string, storageGB *float64) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(storageGB),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr(service),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", usageType))},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}
}

func s3IntelligentTieringStorageCostComponent(name string, region string, usageType string, storageGB *float64) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(storageGB),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr("AmazonS3"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", usageType))},
				{Key: "storageClass", Value: strPtr("Intelligent-Tiering")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}
}

func s3StorageVolumeTypeCostComponent(name string, service string, region string, usageType string, volumeType string, storageGB *float64) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(storageGB),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr(service),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", usageType))},
				{Key: "volumeType", ValueRegex: strPtr(fmt.Sprintf("/%s/i", volumeType))},
				{Key: "operation", Value: strPtr("")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}
}

func s3ApiCostComponent(name string, service string, region string, usageType string, requests *int64) *engine.LineItem {
	return s3ApiOperationCostComponent(name, service, region, usageType, "", requests)
}

func s3ApiOperationCostComponent(name string, service string, region string, usageType string, operation string, requests *int64) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "1k requests",
		UnitMultiplier:  decimal.NewFromInt(1000),
		MonthlyQuantity: intPtrToDecimalPtr(requests),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr(service),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", usageType))},
				{Key: "operation", ValueRegex: strPtr(fmt.Sprintf("/%s/i", operation))},
			},
		},
		UsageBased: true,
	}
}

func s3DataCostComponent(name string, service string, region string, usageType string, dataGB *float64) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(dataGB),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr(service),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", usageType))},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}
}

func s3DataGroupCostComponent(name string, service string, region string, usageType string, group string, dataGB *float64) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(dataGB),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr(service),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", usageType))},
				{Key: "group", ValueRegex: strPtr(fmt.Sprintf("/%s/i", group))},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("0"),
		},
		UsageBased: true,
	}
}

func s3LifecycleTransitionsCostComponent(region string, usageType string, operation string, requests *int64) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Lifecycle transition",
		Unit:            "1k requests",
		UnitMultiplier:  decimal.NewFromInt(1000),
		MonthlyQuantity: intPtrToDecimalPtr(requests),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr("AmazonS3"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s$/i", usageType))},
				{Key: "operation", ValueRegex: strPtr(fmt.Sprintf("/^%s$/i", operation))},
			},
		},
		UsageBased: true,
	}
}

func s3MonitoringCostComponent(region string, objects *int64) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Monitoring and automation",
		Unit:            "1k objects",
		UnitMultiplier:  decimal.NewFromInt(1000),
		MonthlyQuantity: intPtrToDecimalPtr(objects),
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(region),
			Service:    strPtr("AmazonS3"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/Monitoring-Automation-INT/")},
			},
		},
		UsageBased: true,
	}
}
