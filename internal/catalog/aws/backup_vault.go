package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type BackupVault struct {
	Address                       string
	Region                        string
	MonthlyEFSWarmBackupGB        *float64 `c3x_usage:"monthly_efs_warm_backup_gb"`
	MonthlyEFSColdRestoreGB       *float64 `c3x_usage:"monthly_efs_cold_restore_gb"`
	MonthlyRDSSnapshotGB          *float64 `c3x_usage:"monthly_rds_snapshot_gb"`
	MonthlyAuroraSnapshotGB       *float64 `c3x_usage:"monthly_aurora_snapshot_gb"`
	MonthlyDynamodbBackupGB       *float64 `c3x_usage:"monthly_dynamodb_backup_gb"`
	MonthlyDynamodbRestoreGB      *float64 `c3x_usage:"monthly_dynamodb_restore_gb"`
	MonthlyFSxWindowsBackupGB     *float64 `c3x_usage:"monthly_fsx_windows_backup_gb"`
	MonthlyFSxLustreBackupGB      *float64 `c3x_usage:"monthly_fsx_lustre_backup_gb"`
	MonthlyEFSColdBackupGB        *float64 `c3x_usage:"monthly_efs_cold_backup_gb"`
	MonthlyEFSWarmRestoreGB       *float64 `c3x_usage:"monthly_efs_warm_restore_gb"`
	MonthlyEFSItemRestoreRequests *int64   `c3x_usage:"monthly_efs_item_restore_requests"`
	MonthlyEBSSnapshotGB          *float64 `c3x_usage:"monthly_ebs_snapshot_gb"`
}

func (r *BackupVault) CoreType() string {
	return "BackupVault"
}

func (r *BackupVault) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_efs_warm_backup_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_efs_cold_restore_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_rds_snapshot_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_aurora_snapshot_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_dynamodb_backup_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_dynamodb_restore_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_fsx_windows_backup_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_fsx_lustre_backup_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_efs_cold_backup_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_efs_warm_restore_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_efs_item_restore_requests", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_ebs_snapshot_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *BackupVault) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *BackupVault) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	bd := backupData{ref: "monthly_efs_warm_backup_gb", name: "EFS backup (warm)", unit: "GB", usageType: "WarmStorage-ByteHrs-EFS$", service: "AWSBackup", family: "AWS Backup Storage"}
	if r.MonthlyEFSWarmBackupGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyEFSWarmBackupGB))
	}
	costComponents = append(costComponents, r.backupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_efs_cold_backup_gb", name: "EFS backup (cold)", unit: "GB", usageType: "ColdStorage-ByteHrs-EFS$", service: "AWSBackup", family: "AWS Backup Storage"}
	if r.MonthlyEFSColdBackupGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyEFSColdBackupGB))
	}
	costComponents = append(costComponents, r.backupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_efs_warm_restore_gb", name: "EFS restore (warm)", unit: "GB", usageType: "PartialRestore-Warm-EFS", service: "AWSBackup", family: "AWS Backup Storage"}
	if r.MonthlyEFSWarmRestoreGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyEFSWarmRestoreGB))
	}
	costComponents = append(costComponents, r.backupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_efs_cold_restore_gb", name: "EFS restore (cold)", unit: "GB", usageType: "PartialRestore-Cold-EFS", service: "AWSBackup", family: "AWS Backup Storage"}
	if r.MonthlyEFSColdRestoreGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyEFSColdRestoreGB))
	}
	costComponents = append(costComponents, r.backupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_efs_item_restore_requests", name: "EFS restore (item-level)", unit: "requests", usageType: "PartialRestore-Jobs-EFS", service: "AWSBackup", family: "AWS Backup Storage"}
	if r.MonthlyEFSItemRestoreRequests != nil {
		bd.qty = decimalPtr(decimal.NewFromInt(*r.MonthlyEFSItemRestoreRequests))
	}
	costComponents = append(costComponents, r.backupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_ebs_snapshot_gb", name: "EBS snapshot", unit: "GB", usageType: "EBS:SnapshotUsage$", service: "AmazonEC2", family: "Storage Snapshot"}
	if r.MonthlyEBSSnapshotGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyEBSSnapshotGB))
	}
	costComponents = append(costComponents, r.backupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_rds_snapshot_gb", name: "RDS snapshot", unit: "GB", usageType: "RDS:ChargedBackupUsage", service: "AmazonRDS", family: "Storage Snapshot"}
	if r.MonthlyRDSSnapshotGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyRDSSnapshotGB))
	}
	costComponents = append(costComponents, r.backupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_dynamodb_backup_gb", name: "DynamoDB backup", unit: "GB", usageType: "TimedBackupStorage-ByteHrs", service: "AmazonDynamoDB", family: "Amazon DynamoDB On-Demand Backup Storage"}
	if r.MonthlyDynamodbBackupGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyDynamodbBackupGB))
	}
	costComponents = append(costComponents, r.backupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_dynamodb_restore_gb", name: "DynamoDB restore", unit: "GB", usageType: "RestoreDataSize-Bytes", service: "AmazonDynamoDB", family: "Amazon DynamoDB Restore Data Size"}
	if r.MonthlyDynamodbRestoreGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyDynamodbRestoreGB))
	}
	costComponents = append(costComponents, r.backupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_aurora_snapshot_gb", name: "Aurora snapshot", unit: "GB", usageType: "Aurora:BackupUsage", service: "AmazonRDS", family: "Storage Snapshot", key: "databaseEngine", value: "Aurora PostgreSQL"}
	if r.MonthlyAuroraSnapshotGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyAuroraSnapshotGB))
	}
	costComponents = append(costComponents, r.additionalBackupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_fsx_windows_backup_gb", name: "FSx for Windows backup", unit: "GB", usageType: "BackupUsage", service: "AmazonFSx", family: "Storage", key: "fileSystemType", value: "Lustre"}
	if r.MonthlyFSxWindowsBackupGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyFSxWindowsBackupGB))
	}
	costComponents = append(costComponents, r.additionalBackupVaultCostComponent(bd))

	bd = backupData{ref: "monthly_fsx_lustre_backup_gb", name: "FSx for Lustre backup", unit: "GB", usageType: "BackupUsage", service: "AmazonFSx", family: "Storage", key: "fileSystemType", value: "Lustre"}
	if r.MonthlyFSxLustreBackupGB != nil {
		bd.qty = decimalPtr(decimal.NewFromFloat(*r.MonthlyFSxLustreBackupGB))
	}
	costComponents = append(costComponents, r.additionalBackupVaultCostComponent(bd))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents, UsageSchema: r.UsageSchema(),
	}
}

type backupData struct {
	ref       string
	name      string
	unit      string
	usageType string
	service   string
	family    string
	key       string
	value     string
	qty       *decimal.Decimal
}

func (r *BackupVault) backupVaultCostComponent(bd backupData) *engine.LineItem {
	filters := []*engine.AttributeMatch{
		{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", bd.usageType))},
	}

	if bd.name == "RDS snapshot" {
		filters = append(filters, &engine.AttributeMatch{Key: "operation", Value: strPtr("")})
	}

	return &engine.LineItem{
		Name:            bd.name,
		Unit:            bd.unit,
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: bd.qty,
		ProductFilter: &engine.ProductSelector{
			VendorName:       strPtr("aws"),
			Region:           strPtr(r.Region),
			Service:          strPtr(bd.service),
			ProductFamily:    strPtr(bd.family),
			AttributeFilters: filters,
		},
		UsageBased: true,
	}
}

func (r *BackupVault) additionalBackupVaultCostComponent(bd backupData) *engine.LineItem {
	return &engine.LineItem{
		Name:            bd.name,
		Unit:            bd.unit,
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: bd.qty,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr(bd.service),
			ProductFamily: strPtr(bd.family),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s/i", bd.usageType))},
				{Key: bd.key, Value: strPtr(bd.value)},
			},
		},
		UsageBased: true,
	}
}
