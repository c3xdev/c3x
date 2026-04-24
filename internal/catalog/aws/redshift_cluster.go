package aws

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/usage"
)

type RedshiftCluster struct {
	Address                      string
	Region                       string
	NodeType                     string
	Nodes                        *int64
	ManagedStorageGB             *float64 `c3x_usage:"managed_storage_gb"`
	ExcessConcurrencyScalingSecs *int64   `c3x_usage:"excess_concurrency_scaling_secs"`
	SpectrumDataScannedTB        *float64 `c3x_usage:"spectrum_data_scanned_tb"`
	BackupStorageGB              *float64 `c3x_usage:"backup_storage_gb"`
}

func (r *RedshiftCluster) CoreType() string {
	return "RedshiftCluster"
}

func (r *RedshiftCluster) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "managed_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "excess_concurrency_scaling_secs", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "spectrum_data_scanned_tb", ValueType: engine.Float64, DefaultValue: 0.0},
		{Key: "backup_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *RedshiftCluster) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *RedshiftCluster) BuildResource() *engine.Estimate {
	numberOfNodes := int64(1)
	if r.Nodes != nil {
		numberOfNodes = *r.Nodes
	}

	costComponents := []*engine.LineItem{
		{
			Name:           fmt.Sprintf("Cluster usage (%s, %s)", "on-demand", r.NodeType),
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(numberOfNodes)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonRedshift"),
				ProductFamily: strPtr("Compute Instance"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "instanceType", ValueRegex: strPtr(fmt.Sprintf("/%s/i", r.NodeType))},
				},
			},
			PriceFilter: &engine.RateSelector{
				PurchaseOption: strPtr("on_demand"),
			},
		},
	}

	if strings.HasPrefix(r.NodeType, "ra3") {
		var managedStorage *decimal.Decimal
		if r.ManagedStorageGB != nil {
			managedStorage = decimalPtr(decimal.NewFromFloat(*r.ManagedStorageGB))
		}

		costComponents = append(costComponents, r.managedStorageCostComponent(managedStorage))
	}

	if strings.HasPrefix(r.NodeType, "ra3") || strings.HasPrefix(r.NodeType, "ds2") || strings.HasPrefix(r.NodeType, "dc2") {
		var concurrencyScalingSeconds *decimal.Decimal
		if r.ExcessConcurrencyScalingSecs != nil {
			concurrencyScalingSeconds = decimalPtr(decimal.NewFromInt(*r.ExcessConcurrencyScalingSecs))
		}

		costComponents = append(costComponents, r.concurrencyScalingCostComponent(numberOfNodes, concurrencyScalingSeconds))
	}

	var tbScanned *decimal.Decimal
	if r.SpectrumDataScannedTB != nil {
		tbScanned = decimalPtr(decimal.NewFromFloat(*r.SpectrumDataScannedTB))
	}

	costComponents = append(costComponents, r.spectrumCostComponent(tbScanned))

	if r.BackupStorageGB != nil {
		storageSnapshotGB := decimal.NewFromFloat(*r.BackupStorageGB)
		storageSnapshotTiers := usage.CalculateTierBuckets(storageSnapshotGB, []int{51200, 512000})

		if storageSnapshotTiers[0].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, r.storageSnapshotCostComponent("Backup storage (first 50 TB)", "0", &storageSnapshotTiers[0]))
		}

		if storageSnapshotTiers[1].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, r.storageSnapshotCostComponent("Backup storage (next 450 TB)", "51200", &storageSnapshotTiers[1]))
		}

		if storageSnapshotTiers[2].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, r.storageSnapshotCostComponent("Backup storage (over 500 TB)", "512000", &storageSnapshotTiers[2]))
		}
	} else {
		costComponents = append(costComponents, r.storageSnapshotCostComponent("Backup storage (first 50 TB)", "0", nil))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *RedshiftCluster) concurrencyScalingCostComponent(numberOfNodes int64, concurrencySeconds *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            fmt.Sprintf("Concurrency scaling (%s)", r.NodeType),
		Unit:            "node-seconds",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: concurrencySeconds,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonRedshift"),
			ProductFamily: strPtr("Redshift Concurrency Scaling"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "instanceType", ValueRegex: strPtr(fmt.Sprintf("/%s/i", r.NodeType))},
				{Key: "concurrencyscalingfreeusage", Value: strPtr("No")},
			},
		},
		UsageBased: true,
	}
}

func (r *RedshiftCluster) spectrumCostComponent(tbScanned *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Spectrum",
		Unit:            "TB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: tbScanned,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonRedshift"),
			ProductFamily: strPtr("Redshift Data Scan"),
		},
		UsageBased: true,
	}
}

func (r *RedshiftCluster) storageSnapshotCostComponent(displayName string, startUsageAmount string, storageSnapshot *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            displayName,
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: storageSnapshot,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonRedshift"),
			ProductFamily: strPtr("Storage Snapshot"),
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(startUsageAmount),
		},
		UsageBased: true,
	}
}

func (r *RedshiftCluster) managedStorageCostComponent(managedStorage *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            fmt.Sprintf("Managed storage (%s)", r.NodeType),
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: managedStorage,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonRedshift"),
			ProductFamily: strPtr("Redshift Managed Storage"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "instanceType", ValueRegex: strPtr(fmt.Sprintf("/%s/i", r.NodeType))},
			},
		},
		UsageBased: true,
	}
}
