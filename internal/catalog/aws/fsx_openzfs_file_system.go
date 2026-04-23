package aws

import (
	"fmt"
	"math"
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type FSxOpenZFSFileSystem struct {
	Address                   string
	StorageType               string
	ThroughputCapacity        int64
	ProvisionedIOPS           int64
	ProvisionedIOPSMode       string
	StorageCapacityGB         int64
	Region                    string
	DeploymentType            string
	DataCompression           string
	CompressionSavingsPercent *float64 `c3x_usage:"compression_savings_percent"`
	BackupStorageGB           *float64 `c3x_usage:"backup_storage_gb"`
}

func (r *FSxOpenZFSFileSystem) CoreType() string {
	return "FSxOpenZFSFileSystem"
}

func (r *FSxOpenZFSFileSystem) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "backup_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *FSxOpenZFSFileSystem) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *FSxOpenZFSFileSystem) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			r.throughputCapacityCostComponent(),
			r.provisionedIOPSCapacityCostComponent(),
			r.storageCapacityCostComponent(),
			r.backupGBCostComponent(),
		},
		UsageSchema: r.UsageSchema(),
	}
}

func (r *FSxOpenZFSFileSystem) throughputCapacityCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Throughput capacity",
		Unit:            "MBps",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(r.ThroughputCapacity)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonFSx"),
			ProductFamily: strPtr("Provisioned Throughput"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "deploymentOption", Value: strPtr("Single-AZ")},
				{Key: "fileSystemType", Value: strPtr("OpenZFS")},
			},
		},
	}
}

func (r *FSxOpenZFSFileSystem) provisionedIOPSCapacityCostComponent() *engine.LineItem {
	var provisionedIOPS = decimalPtr(decimal.NewFromInt(0))
	if r.ProvisionedIOPSMode == "USER_PROVISIONED" {
		provisionedIOPS = decimalPtr(decimal.NewFromFloat(math.Max(0, float64(r.ProvisionedIOPS-(3*r.StorageCapacityGB)))))
	}
	return &engine.LineItem{
		Name:            "Provisioned IOPS",
		Unit:            "IOPS",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: provisionedIOPS,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonFSx"),
			ProductFamily: strPtr("Provisioned IOPS"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "deploymentOption", Value: strPtr("Single-AZ")},
				{Key: "fileSystemType", Value: strPtr("OpenZFS")},
			},
		},
	}
}

func (r *FSxOpenZFSFileSystem) storageCapacityCostComponent() *engine.LineItem {
	var storageCapacity *decimal.Decimal
	var compressionEnabled = ""
	var compressionSavingsPercent float64
	if r.DataCompression != "" && r.DataCompression != "NONE" {
		if r.CompressionSavingsPercent != nil {
			compressionSavingsPercent = *r.CompressionSavingsPercent
		} else {
			compressionSavingsPercent = 50
		}
		compressionEnabled = fmt.Sprintf(" (%s compression, %.0f%%)", r.DataCompression, compressionSavingsPercent)
		storageCapacity = decimalPtr(decimal.NewFromFloat(math.Ceil(float64(r.StorageCapacityGB) * (1 - compressionSavingsPercent/100))))
	} else {
		storageCapacity = decimalPtr(decimal.NewFromInt(r.StorageCapacityGB))
	}

	return &engine.LineItem{
		Name:            fmt.Sprintf("SSD storage%s", compressionEnabled),
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: storageCapacity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonFSx"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "deploymentOption", Value: strPtr("Single-AZ")},
				{Key: "fileSystemType", Value: strPtr("OpenZFS")},
				{Key: "storageType", Value: strPtr("SSD")},
			},
		},
	}
}

func (r *FSxOpenZFSFileSystem) backupGBCostComponent() *engine.LineItem {
	var backupStorage *decimal.Decimal
	if r.BackupStorageGB != nil {
		backupStorage = decimalPtr(decimal.NewFromFloat(*r.BackupStorageGB))
	}

	filters := []*engine.AttributeMatch{
		{Key: "usagetype", ValueRegex: strPtr("/BackupUsage/")},
		{Key: "fileSystemType", Value: strPtr("OpenZFS")},
	}
	if strings.Contains(strings.ToLower(r.DeploymentType), "multi") {
		filters = append(filters, &engine.AttributeMatch{
			Key:   "deploymentOption",
			Value: strPtr("Multi-AZ"),
		})
	} else {
		filters = append(filters, &engine.AttributeMatch{
			Key:   "deploymentOption",
			Value: strPtr("N/A"),
		})
	}

	return &engine.LineItem{
		Name:            "Backup storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: backupStorage,
		ProductFilter: &engine.ProductSelector{
			VendorName:       strPtr("aws"),
			Region:           strPtr(r.Region),
			Service:          strPtr("AmazonFSx"),
			ProductFamily:    strPtr("Storage"),
			AttributeFilters: filters,
		},
		UsageBased: true,
	}
}
