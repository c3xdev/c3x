package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type S3StandardStorageClass struct {
	// "required" args that can't really be missing.
	Region string

	// "usage" args
	StorageGB                   *float64 `c3x_usage:"storage_gb"`
	MonthlyTier1Requests        *int64   `c3x_usage:"monthly_tier_1_requests"`
	MonthlyTier2Requests        *int64   `c3x_usage:"monthly_tier_2_requests"`
	MonthlySelectDataScannedGB  *float64 `c3x_usage:"monthly_select_data_scanned_gb"`
	MonthlySelectDataReturnedGB *float64 `c3x_usage:"monthly_select_data_returned_gb"`
}

var S3StandardStorageClassUsageSchema = []*engine.ConsumptionField{
	{Key: "storage_gb", DefaultValue: 0.0, ValueType: engine.Float64},
	{Key: "monthly_tier_1_requests", DefaultValue: 0, ValueType: engine.Int64},
	{Key: "monthly_tier_2_requests", DefaultValue: 0, ValueType: engine.Int64},
	{Key: "monthly_select_data_scanned_gb", DefaultValue: 0, ValueType: engine.Float64},
	{Key: "monthly_select_data_returned_gb", DefaultValue: 0, ValueType: engine.Float64},
}

func (a *S3StandardStorageClass) CoreType() string {
	return "S3StandardStorageClass"
}

func (a *S3StandardStorageClass) UsageSchema() []*engine.ConsumptionField {
	return S3StandardStorageClassUsageSchema
}

func (a *S3StandardStorageClass) UsageKey() string {
	return "standard"
}

func (a *S3StandardStorageClass) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(a, u)
}

func (a *S3StandardStorageClass) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:        "Standard",
		UsageSchema: a.UsageSchema(),
		CostComponents: []*engine.LineItem{
			s3StorageVolumeTypeCostComponent("Storage", "AmazonS3", a.Region, "TimedStorage-ByteHrs", "Standard", a.StorageGB),
			s3ApiCostComponent("PUT, COPY, POST, LIST requests", "AmazonS3", a.Region, "Requests-Tier1", a.MonthlyTier1Requests),
			s3ApiCostComponent("GET, SELECT, and all other requests", "AmazonS3", a.Region, "Requests-Tier2", a.MonthlyTier2Requests),
			s3DataGroupCostComponent("Select data scanned", "AmazonS3", a.Region, "Select-Scanned-Bytes", "S3-API-Select-Scanned", a.MonthlySelectDataScannedGB),
			s3DataGroupCostComponent("Select data returned", "AmazonS3", a.Region, "Select-Returned-Bytes", "S3-API-Select-Returned", a.MonthlySelectDataReturnedGB),
		},
	}
}
