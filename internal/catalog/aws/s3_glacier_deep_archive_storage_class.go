package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type S3GlacierDeepArchiveStorageClass struct {
	// "required" args that can't really be missing.
	Region string

	// "usage" args
	StorageGB                            *float64 `c3x_usage:"storage_gb"`
	MonthlyTier1Requests                 *int64   `c3x_usage:"monthly_tier_1_requests"`
	MonthlyTier2Requests                 *int64   `c3x_usage:"monthly_tier_2_requests"`
	MonthlyLifecycleTransitionRequests   *int64   `c3x_usage:"monthly_lifecycle_transition_requests"`
	MonthlyStandardDataRetrievalRequests *int64   `c3x_usage:"monthly_standard_data_retrieval_requests"`
	MonthlyStandardDataRetrievalGB       *float64 `c3x_usage:"monthly_standard_data_retrieval_gb"`
	MonthlyBulkDataRetrievalRequests     *int64   `c3x_usage:"monthly_bulk_data_retrieval_requests"`
	MonthlyBulkDataRetrievalGB           *float64 `c3x_usage:"monthly_bulk_data_retrieval_gb"`
	EarlyDeleteGB                        *float64 `c3x_usage:"early_delete_gb"`
}

var S3GlacierDeepArchiveStorageClassUsageSchema = []*engine.ConsumptionField{
	{Key: "storage_gb", DefaultValue: 0.0, ValueType: engine.Float64},
	{Key: "monthly_tier_1_requests", DefaultValue: 0, ValueType: engine.Int64},
	{Key: "monthly_tier_2_requests", DefaultValue: 0, ValueType: engine.Int64},
	{Key: "monthly_lifecycle_transition_requests", DefaultValue: 0, ValueType: engine.Int64},
	{Key: "monthly_standard_data_retrieval_requests", DefaultValue: 0, ValueType: engine.Int64},
	{Key: "monthly_standard_data_retrieval_gb", DefaultValue: 0.0, ValueType: engine.Float64},
	{Key: "monthly_bulk_data_retrieval_requests", DefaultValue: 0, ValueType: engine.Int64},
	{Key: "monthly_bulk_data_retrieval_gb", DefaultValue: 0.0, ValueType: engine.Float64},
	{Key: "early_delete_gb", DefaultValue: 0.0, ValueType: engine.Float64},
}

func (a *S3GlacierDeepArchiveStorageClass) CoreType() string {
	return "S3GlacierDeepArchiveStorageClass"
}

func (a *S3GlacierDeepArchiveStorageClass) UsageSchema() []*engine.ConsumptionField {
	return S3GlacierDeepArchiveStorageClassUsageSchema
}

func (a *S3GlacierDeepArchiveStorageClass) UsageKey() string {
	return "glacier_deep_archive"
}

func (a *S3GlacierDeepArchiveStorageClass) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(a, u)
}

func (a *S3GlacierDeepArchiveStorageClass) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name:        "Glacier deep archive",
		UsageSchema: a.UsageSchema(),
		CostComponents: []*engine.LineItem{
			s3StorageCostComponent("Storage", "AmazonS3GlacierDeepArchive", a.Region, "TimedStorage-GDA-ByteHrs", a.StorageGB),
			s3ApiOperationCostComponent("PUT, COPY, POST, LIST requests", "AmazonS3GlacierDeepArchive", a.Region, "Requests-GDA-Tier1", "PostObject", a.MonthlyTier1Requests),
			s3ApiCostComponent("GET, SELECT, and all other requests", "AmazonS3", a.Region, "Requests-GDA-Tier2", a.MonthlyTier2Requests),
			s3LifecycleTransitionsCostComponent(a.Region, "Requests-Tier3", "S3-GDATransition", a.MonthlyLifecycleTransitionRequests),
			s3ApiOperationCostComponent("Retrieval requests (standard)", "AmazonS3GlacierDeepArchive", a.Region, "Requests-GDA-Tier3", "", a.MonthlyStandardDataRetrievalRequests),
			s3DataCostComponent("Retrievals (standard)", "AmazonS3GlacierDeepArchive", a.Region, "Standard-Retrieval-Bytes", a.MonthlyStandardDataRetrievalGB),
			s3ApiCostComponent("Retrieval requests (bulk)", "AmazonS3GlacierDeepArchive", a.Region, "Requests-GDA-Tier5", a.MonthlyBulkDataRetrievalRequests),
			s3DataCostComponent("Retrievals (bulk)", "AmazonS3GlacierDeepArchive", a.Region, "Bulk-Retrieval-Bytes", a.MonthlyBulkDataRetrievalGB),
			s3DataCostComponent("Early delete (within 180 days)", "AmazonS3GlacierDeepArchive", a.Region, "EarlyDelete-GDA", a.EarlyDeleteGB),
		},
	}
}
