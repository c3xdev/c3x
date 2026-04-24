package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"

	"fmt"
	"math"
	"strings"

	"github.com/shopspring/decimal"
)

const Standard = "Standard"
const StandardSSD = "StandardSSD"
const Premium = "Premium"

type ManagedDisk struct {
	Address string
	Region  string
	ManagedDiskData
	MonthlyDiskOperations *int64 `c3x_usage:"monthly_disk_operations"`
}

type ManagedDiskData struct {
	DiskType          string
	DiskSizeGB        int64
	DiskIOPSReadWrite int64
	DiskMBPSReadWrite int64
}

func (r *ManagedDisk) CoreType() string {
	return "ManagedDisk"
}

func (r *ManagedDisk) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_disk_operations", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *ManagedDisk) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ManagedDisk) BuildResource() *engine.Estimate {
	region := r.Region
	diskType := r.DiskType

	var monthlyDiskOperations *decimal.Decimal

	if r.MonthlyDiskOperations != nil {
		monthlyDiskOperations = decimalPtr(decimal.NewFromInt(*r.MonthlyDiskOperations))
	}

	costComponents := managedDiskCostComponents(region, diskType, r.DiskSizeGB, r.DiskIOPSReadWrite, r.DiskMBPSReadWrite, monthlyDiskOperations)

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

var diskSizeMap = map[string][]struct {
	Name string
	Size int
}{

	"Standard": {
		{"S4", 32},
		{"S6", 64},
		{"S10", 128},
		{"S15", 256},
		{"S20", 512},
		{"S30", 1024},
		{"S40", 2048},
		{"S50", 4096},
		{"S60", 8192},
		{"S70", 16384},
		{"S80", 32767},
	},
	"StandardSSD": {
		{"E1", 4},
		{"E2", 8},
		{"E3", 16},
		{"E4", 32},
		{"E6", 64},
		{"E10", 128},
		{"E15", 256},
		{"E20", 512},
		{"E30", 1024},
		{"E40", 2048},
		{"E50", 4096},
		{"E60", 8192},
		{"E70", 16384},
		{"E80", 32767},
	},
	"Premium": {
		{"P1", 4},
		{"P2", 8},
		{"P3", 16},
		{"P4", 32},
		{"P6", 64},
		{"P10", 128},
		{"P15", 256},
		{"P20", 512},
		{"P30", 1024},
		{"P40", 2048},
		{"P50", 4096},
		{"P60", 8192},
		{"P70", 16384},
		{"P80", 32767},
	},
}

var storageReplicationTypes = []string{"LRS", "ZRS"}
var ultraDiskSizes = []int{4, 8, 16, 32, 64, 128, 256, 512}
var ultraDiskSizeStep = 1024
var ultraDiskMaxSize = 65536

var diskProductNameMap = map[string]string{
	"Standard":    "Standard HDD Managed Disks",
	"StandardSSD": "Standard SSD Managed Disks",
	"Premium":     "Premium SSD Managed Disks",
}

func managedDiskCostComponents(region, diskType string, diskSizeGB, diskIOPSReadWrite, diskMBPSReadWrite int64, monthlyDiskOperations *decimal.Decimal) []*engine.LineItem {
	p := strings.Split(diskType, "_")
	diskTypePrefix := p[0]

	var storageReplicationType string
	if len(p) > 1 {
		storageReplicationType = strings.ToUpper(p[1])
	}

	validstorageReplicationType := mapStorageReplicationType(storageReplicationType)
	if !validstorageReplicationType {
		logging.Logger.Warn().Msgf("Could not map %s to a valid storage type", storageReplicationType)
		return nil
	}

	if strings.ToLower(diskTypePrefix) == "ultrassd" {
		return ultraDiskCostComponents(region, storageReplicationType, diskSizeGB, diskIOPSReadWrite, diskMBPSReadWrite)
	}

	return standardPremiumDiskCostComponents(region, diskTypePrefix, storageReplicationType, diskSizeGB, monthlyDiskOperations)
}

func standardPremiumDiskCostComponents(region string, diskTypePrefix string, storageReplicationType string, diskSizeGB int64, monthlyDiskOperations *decimal.Decimal) []*engine.LineItem {
	requestedSize := 30

	if diskSizeGB > 0 {
		requestedSize = int(diskSizeGB)
	}

	diskName := mapDiskName(diskTypePrefix, requestedSize)
	if diskName == "" {
		logging.Logger.Warn().Msgf("Could not map disk type %s and size %d to disk name", diskTypePrefix, requestedSize)
		return nil
	}

	productName, ok := diskProductNameMap[diskTypePrefix]
	if !ok {
		logging.Logger.Warn().Msgf("Could not map disk type %s to product name", diskTypePrefix)
		return nil
	}

	costComponents := []*engine.LineItem{storageCostComponent(region, diskName, storageReplicationType, productName)}

	if strings.ToLower(diskTypePrefix) == "standard" || strings.ToLower(diskTypePrefix) == "standardssd" {
		var opsQty *decimal.Decimal

		if monthlyDiskOperations != nil {
			opsQty = decimalPtr(monthlyDiskOperations.Div(decimal.NewFromInt(10000)))
		}

		costComponents = append(costComponents, &engine.LineItem{
			Name:            "Disk operations",
			Unit:            "10k operations",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: opsQty,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(region),
				Service:       strPtr("Storage"),
				ProductFamily: strPtr("Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "productName", Value: strPtr(productName)},
					{Key: "skuName", Value: strPtr(fmt.Sprintf("%s %s", diskName, storageReplicationType))},
					{Key: "meterName", ValueRegex: regexPtr("Disk Operations$")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
			UsageBased: true,
		})
	}

	return costComponents
}

func storageCostComponent(region, diskName, storageReplicationType, productName string) *engine.LineItem {
	return &engine.LineItem{
		Name:            fmt.Sprintf("Storage (%s, %s)", diskName, storageReplicationType),
		Unit:            "months",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(region),
			Service:       strPtr("Storage"),
			ProductFamily: strPtr("Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "productName", Value: strPtr(productName)},
				{Key: "skuName", Value: strPtr(fmt.Sprintf("%s %s", diskName, storageReplicationType))},
				{Key: "meterName", ValueRegex: regexPtr(fmt.Sprintf("^%s (%s )?Disk(s)?$", diskName, storageReplicationType))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func ultraDiskCostComponents(region string, storageReplicationType string, diskSizeGB, diskIOPSReadWrite, diskMBPSReadWrite int64) []*engine.LineItem {
	requestedSize := 1024
	iops := 2048
	throughput := 8

	if diskSizeGB > 0 {
		requestedSize = int(diskSizeGB)
	}

	if diskIOPSReadWrite > 0 {
		iops = int(diskIOPSReadWrite)
	}

	if diskMBPSReadWrite > 0 {
		throughput = int(diskMBPSReadWrite)
	}

	diskSize := mapUltraDiskSize(requestedSize)

	costComponents := []*engine.LineItem{
		{
			Name:           fmt.Sprintf("Storage (ultra, %d GiB)", diskSize),
			Unit:           "GiB",
			UnitMultiplier: engine.HourToMonthUnitMultiplier,
			HourlyQuantity: decimalPtr(decimal.NewFromInt(int64(diskSize))),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(region),
				Service:       strPtr("Storage"),
				ProductFamily: strPtr("Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "productName", Value: strPtr("Ultra Disks")},
					{Key: "skuName", Value: strPtr(fmt.Sprintf("Ultra %s", storageReplicationType))},
					{Key: "meterName", ValueRegex: regexPtr("Provisioned Capacity$")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		},
		{
			Name:           "Provisioned IOPS",
			Unit:           "IOPS",
			UnitMultiplier: engine.HourToMonthUnitMultiplier,
			HourlyQuantity: decimalPtr(decimal.NewFromInt(int64(iops))),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(region),
				Service:       strPtr("Storage"),
				ProductFamily: strPtr("Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "productName", Value: strPtr("Ultra Disks")},
					{Key: "skuName", Value: strPtr(fmt.Sprintf("Ultra %s", storageReplicationType))},
					{Key: "meterName", ValueRegex: regexPtr("Provisioned IOPS$")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		},
		{
			Name:           "Throughput",
			Unit:           "MB/s",
			UnitMultiplier: engine.HourToMonthUnitMultiplier,
			HourlyQuantity: decimalPtr(decimal.NewFromInt(int64(throughput))),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("azure"),
				Region:        strPtr(region),
				Service:       strPtr("Storage"),
				ProductFamily: strPtr("Storage"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "productName", Value: strPtr("Ultra Disks")},
					{Key: "skuName", Value: strPtr(fmt.Sprintf("Ultra %s", storageReplicationType))},
					{Key: "meterName", ValueRegex: regexPtr("Provisioned Throughput \\(MBps\\)$")},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("Consumption"),
			},
		},
	}

	return costComponents
}

func mapDiskName(diskType string, requestedSize int) string {
	diskTypeMap, ok := diskSizeMap[diskType]
	if !ok {
		return ""
	}

	name := ""
	for _, v := range diskTypeMap {
		name = v.Name
		if v.Size >= requestedSize {
			break
		}
	}

	if requestedSize > diskTypeMap[len(diskTypeMap)-1].Size {
		return ""
	}

	return name
}

func mapStorageReplicationType(storageReplicationType string) bool {
	for _, b := range storageReplicationTypes {
		if storageReplicationType == b {
			return true
		}
	}

	return false
}

func mapUltraDiskSize(requestedSize int) int {
	if requestedSize >= ultraDiskMaxSize {
		return ultraDiskMaxSize
	}

	if requestedSize < ultraDiskSizes[0] {
		return ultraDiskSizes[0]
	}

	if requestedSize > ultraDiskSizes[len(ultraDiskSizes)-1] {
		return int(math.Ceil(float64(requestedSize)/float64(ultraDiskSizeStep))) * ultraDiskSizeStep
	}

	size := 0
	for _, v := range ultraDiskSizes {
		size = v
		if size >= requestedSize {
			break
		}
	}

	return size

}
