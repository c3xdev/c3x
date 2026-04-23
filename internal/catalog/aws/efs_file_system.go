package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type EFSFileSystem struct {
	Address                        string
	Region                         string
	HasLifecyclePolicy             bool
	AvailabilityZoneName           string
	ProvisionedThroughputInMBps    float64
	InfrequentAccessStorageGB      *float64 `c3x_usage:"infrequent_access_storage_gb"`
	StorageGB                      *float64 `c3x_usage:"storage_gb"`
	MonthlyInfrequentAccessReadGB  *float64 `c3x_usage:"monthly_infrequent_access_read_gb"`
	MonthlyInfrequentAccessWriteGB *float64 `c3x_usage:"monthly_infrequent_access_write_gb"`
}

func (r *EFSFileSystem) CoreType() string {
	return "EFSFileSystem"
}

func (r *EFSFileSystem) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "infrequent_access_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_infrequent_access_read_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_infrequent_access_write_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *EFSFileSystem) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EFSFileSystem) BuildResource() *engine.Estimate {
	costComponents := make([]*engine.LineItem, 0)

	var storageGB *decimal.Decimal
	if r.StorageGB != nil {
		storageGB = decimalPtr(decimal.NewFromFloat(*r.StorageGB))
	}

	if r.AvailabilityZoneName != "" {
		costComponents = append(costComponents, r.storageCostComponent("Storage (one zone)", "-TimedStorage-Z-ByteHrs", storageGB))
	} else {
		costComponents = append(costComponents, r.storageCostComponent("Storage (standard)", "-TimedStorage-ByteHrs", storageGB))
	}

	if r.ProvisionedThroughputInMBps > 0 {
		provisionedThroughput := r.calculateProvisionedThroughput(storageGB, decimal.NewFromFloat(r.ProvisionedThroughputInMBps))
		costComponents = append(costComponents, r.provisionedThroughputCostComponent(provisionedThroughput))
	}

	if r.HasLifecyclePolicy {
		var infrequentAccessStorageGB *decimal.Decimal
		if r.InfrequentAccessStorageGB != nil {
			infrequentAccessStorageGB = decimalPtr(decimal.NewFromFloat(*r.InfrequentAccessStorageGB))
		}

		if r.AvailabilityZoneName != "" {
			costComponents = append(costComponents, r.storageCostComponent("Storage (one zone, infrequent access)", "IATimedStorage-Z-ByteHrs", infrequentAccessStorageGB))
		} else {
			costComponents = append(costComponents, r.storageCostComponent("Storage (standard, infrequent access)", "-IATimedStorage-ByteHrs", infrequentAccessStorageGB))
		}

		var infrequentAccessReadGB *decimal.Decimal
		if r.MonthlyInfrequentAccessReadGB != nil {
			infrequentAccessReadGB = decimalPtr(decimal.NewFromFloat(*r.MonthlyInfrequentAccessReadGB))
		}

		var infrequentAccessWriteGB *decimal.Decimal
		if r.MonthlyInfrequentAccessWriteGB != nil {
			infrequentAccessWriteGB = decimalPtr(decimal.NewFromFloat(*r.MonthlyInfrequentAccessWriteGB))
		}

		costComponents = append(costComponents, r.requestsCostComponent("Read requests (infrequent access)", "Read", infrequentAccessReadGB))
		costComponents = append(costComponents, r.requestsCostComponent("Write requests (infrequent access)", "Write", infrequentAccessWriteGB))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *EFSFileSystem) calculateProvisionedThroughput(storageGB *decimal.Decimal, throughput decimal.Decimal) *decimal.Decimal {
	if storageGB == nil {
		storageGB = &decimal.Zero
	}

	defaultThroughput := storageGB.Mul(decimal.NewFromInt(730).Div(decimal.NewFromInt(20).Mul(decimal.NewFromInt(1))))
	totalProvisionedThroughput := throughput.Mul(decimal.NewFromInt(730))
	totalBillableProvisionedThroughput := totalProvisionedThroughput.Sub(defaultThroughput).Div(decimal.NewFromInt(730))

	if totalBillableProvisionedThroughput.IsPositive() {
		return &totalBillableProvisionedThroughput
	}

	return &decimal.Zero
}

func (r *EFSFileSystem) storageCostComponent(name, usagetype string, storageGB *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: storageGB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonEFS"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", usagetype))},
			},
		},
		UsageBased: true,
	}
}

func (r *EFSFileSystem) provisionedThroughputCostComponent(provisionedThroughputMiBps *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Provisioned throughput",
		Unit:            "MBps",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: provisionedThroughputMiBps,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonEFS"),
			ProductFamily: strPtr("Provisioned Throughput"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/ProvisionedTP-MiBpsHrs/")},
			},
		},
		UsageBased: true,
	}
}

func (r *EFSFileSystem) requestsCostComponent(name, accessType string, requestsGB *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: requestsGB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonEFS"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "accessType", Value: strPtr(accessType)},
				{Key: "storageClass", Value: strPtr("Infrequent Access")},
			},
		},
		UsageBased: true,
	}
}
