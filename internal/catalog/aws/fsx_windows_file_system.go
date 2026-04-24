package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

type FSxWindowsFileSystem struct {
	Address            string
	StorageType        string
	ThroughputCapacity int64
	StorageCapacityGB  int64
	Region             string
	DeploymentType     string
	BackupStorageGB    *float64 `c3x_usage:"backup_storage_gb"`
}

func (r *FSxWindowsFileSystem) CoreType() string {
	return "FSxWindowsFileSystem"
}

func (r *FSxWindowsFileSystem) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "backup_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *FSxWindowsFileSystem) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *FSxWindowsFileSystem) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			r.throughputCapacityCostComponent(),
			r.storageCapacityCostComponent(),
			r.backupGBCostComponent(),
		},
		UsageSchema: r.UsageSchema(),
	}
}

func (r *FSxWindowsFileSystem) deploymentOptionValue() string {
	if strings.Contains(strings.ToLower(r.DeploymentType), "multi_az") {
		return "Multi-AZ"
	}

	return "Single-AZ"
}

func (r *FSxWindowsFileSystem) storageTypeValue() string {
	if strings.ToLower(r.StorageType) == "hdd" {
		return "HDD"
	}

	return "SSD"
}

func (r *FSxWindowsFileSystem) throughputCapacityCostComponent() *engine.LineItem {
	deploymentOption := r.deploymentOptionValue()

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
				{Key: "deploymentOption", Value: strPtr(deploymentOption)},
				{Key: "fileSystemType", Value: strPtr("Windows")},
			},
		},
	}
}

func (r *FSxWindowsFileSystem) storageCapacityCostComponent() *engine.LineItem {
	deploymentOption := r.deploymentOptionValue()
	storageType := r.storageTypeValue()

	return &engine.LineItem{
		Name:            fmt.Sprintf("%v storage", storageType),
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(r.StorageCapacityGB)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonFSx"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "deploymentOption", Value: strPtr(deploymentOption)},
				{Key: "fileSystemType", Value: strPtr("Windows")},
				{Key: "storageType", Value: strPtr(storageType)},
			},
		},
	}
}

func (r *FSxWindowsFileSystem) backupGBCostComponent() *engine.LineItem {
	deploymentOption := r.deploymentOptionValue()

	var backupStorage *decimal.Decimal
	if r.BackupStorageGB != nil {
		backupStorage = decimalPtr(decimal.NewFromFloat(*r.BackupStorageGB))
	}

	return &engine.LineItem{
		Name:            "Backup storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: backupStorage,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonFSx"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "deploymentOption", Value: strPtr(deploymentOption)},
				{Key: "usagetype", ValueRegex: strPtr("/BackupUsage/")},
				{Key: "fileSystemType", Value: strPtr("Windows")},
			},
		},
		UsageBased: true,
	}
}
