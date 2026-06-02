package aws_test

import (
	"testing"
	"time"

	resources "github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/stretchr/testify/assert"
)

func stubListBucketMetricsConfigurations(stub *stubbedAWS) {
	stub.WhenFullPath("/test-bucket?metrics=&x-id=ListBucketMetricsConfigurations").Then(200, `
		<ListMetricsConfigurationsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
			<MetricsConfiguration>
				<Filter>
					<Prefix>test-prefix</Prefix>
					<Id>withPrefix</Id>
				</Filter>
			</MetricsConfiguration>
			<MetricsConfiguration>
				<Id>c3x</Id>
			</MetricsConfiguration>
			<IsTruncated>false</IsTruncated>
		</ListMetricsConfigurationsResult>`)
}

func stubListBucketMetricsConfigurationsNoMatching(stub *stubbedAWS) {
	stub.WhenFullPath("/test-bucket?metrics=&x-id=ListBucketMetricsConfigurations").Then(200, `
		<ListMetricsConfigurationsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
			<MetricsConfiguration>
				<Filter>
					<Prefix>test-prefix</Prefix>
					<Id>withPrefix</Id>
				</Filter>
			</MetricsConfiguration>
			<IsTruncated>false</IsTruncated>
		</ListMetricsConfigurationsResult>`)
}

func stubStorageClassBytes(stub *stubbedAWS, storageClass string, bytes int) {
	stub.WhenBody(storageClass, "BucketSizeBytes", "Average", "Bytes").
		OnPathPrefix(cloudWatchSmithyPathPrefix + "GetMetricStatistics").
		ThenCBOR(200, map[string]interface{}{
			"Label": "BucketSizeBytes",
			"Datapoints": []interface{}{
				map[string]interface{}{
					"Unit":      "Bytes",
					"Average":   float64(bytes),
					"Timestamp": time.Unix(0, 0).UTC(),
				},
			},
		})
}

func stubRequestCounts(stub *stubbedAWS, metric string, count int) {
	stub.WhenBody("c3x", metric, "Sum", "Count").
		OnPathPrefix(cloudWatchSmithyPathPrefix + "GetMetricStatistics").
		ThenCBOR(200, map[string]interface{}{
			"Label": metric,
			"Datapoints": []interface{}{
				map[string]interface{}{
					"Unit":      "Count",
					"Sum":       float64(count),
					"Timestamp": time.Unix(0, 0).UTC(),
				},
			},
		})
}

func stubDataBytes(stub *stubbedAWS, metric string, bytes int) {
	stub.WhenBody("c3x", metric, "Sum", "Bytes").
		OnPathPrefix(cloudWatchSmithyPathPrefix + "GetMetricStatistics").
		ThenCBOR(200, map[string]interface{}{
			"Label": metric,
			"Datapoints": []interface{}{
				map[string]interface{}{
					"Unit":      "Bytes",
					"Sum":       float64(bytes),
					"Timestamp": time.Unix(0, 0).UTC(),
				},
			},
		})
}

func TestS3Bucket(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()

	stubListBucketMetricsConfigurations(stub)

	storageClassBytes := map[string]int{
		"StandardStorage":              2100000000,
		"IntelligentTieringFAStorage":  2200000000,
		"IntelligentTieringIAStorage":  2300000000,
		"IntelligentTieringAAStorage":  2400000000,
		"IntelligentTieringDAAStorage": 2500000000,
		"StandardIAStorage":            2600000000,
		"OneZoneIAStorage":             2700000000,
		"GlacierStorage":               2800000000,
		"DeepArchiveStorage":           0, // This should not appear in estimates.usages
	}

	for storageClass, bytes := range storageClassBytes {
		stubStorageClassBytes(stub, storageClass, bytes)
	}

	requestCounts := map[string]int{
		"PutRequests":    100,
		"PostRequests":   200,
		"ListRequests":   300,
		"GetRequests":    400,
		"HeadRequests":   500,
		"SelectRequests": 600,
	}

	for metric, count := range requestCounts {
		stubRequestCounts(stub, metric, count)
	}

	dataBytes := map[string]int{
		"SelectBytesScanned":  1100000000,
		"SelectBytesReturned": 1200000000,
	}

	for metric, bytes := range dataBytes {
		stubDataBytes(stub, metric, bytes)
	}

	args := resources.S3Bucket{
		Name:   "test-bucket",
		Region: "us-east-1",
	}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)

	assert.Equal(t, map[string]interface{}{
		"standard": map[string]interface{}{
			"storage_gb":                      2.1,
			"monthly_tier_1_requests":         int64(600),
			"monthly_tier_2_requests":         int64(1500),
			"monthly_select_data_scanned_gb":  1.1,
			"monthly_select_data_returned_gb": 1.2,
		},
		"intelligent_tiering": map[string]interface{}{
			"frequent_access_storage_gb":     2.2,
			"infrequent_access_storage_gb":   2.3,
			"archive_access_storage_gb":      2.4,
			"deep_archive_access_storage_gb": 2.5,
		},
		"standard_infrequent_access": map[string]interface{}{
			"storage_gb": 2.6,
		},
		"one_zone_infrequent_access": map[string]interface{}{
			"storage_gb": 2.7,
		},
		"glacier_flexible_retrieval": map[string]interface{}{
			"storage_gb": 2.8,
		},
	}, estimates.usage)
}

func TestS3BucketNoFilter(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()

	stubListBucketMetricsConfigurationsNoMatching(stub)

	storageClassBytes := map[string]int{
		"StandardStorage":              2100000000,
		"IntelligentTieringFAStorage":  2200000000,
		"IntelligentTieringIAStorage":  2300000000,
		"IntelligentTieringAAStorage":  2400000000,
		"IntelligentTieringDAAStorage": 2500000000,
		"StandardIAStorage":            2600000000,
		"OneZoneIAStorage":             2700000000,
		"GlacierStorage":               2800000000,
		"DeepArchiveStorage":           2900000000,
	}

	for storageClass, bytes := range storageClassBytes {
		stubStorageClassBytes(stub, storageClass, bytes)
	}

	args := resources.S3Bucket{
		Name:   "test-bucket",
		Region: "us-east-1",
	}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)

	assert.Equal(t, map[string]interface{}{
		"storage_gb": 2.1,
	}, estimates.usage["standard"])
}

func TestS3BucketNoStandard(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()

	stubListBucketMetricsConfigurations(stub)

	storageClassBytes := map[string]int{
		"StandardStorage":              0,
		"IntelligentTieringFAStorage":  2200000000,
		"IntelligentTieringIAStorage":  0,
		"IntelligentTieringAAStorage":  0,
		"IntelligentTieringDAAStorage": 0,
		"StandardIAStorage":            0,
		"OneZoneIAStorage":             0,
		"GlacierStorage":               0,
		"DeepArchiveStorage":           0,
	}

	for storageClass, bytes := range storageClassBytes {
		stubStorageClassBytes(stub, storageClass, bytes)
	}

	requestCounts := map[string]int{
		"PutRequests":    100,
		"PostRequests":   200,
		"ListRequests":   300,
		"GetRequests":    400,
		"HeadRequests":   500,
		"SelectRequests": 600,
	}

	for metric, count := range requestCounts {
		stubRequestCounts(stub, metric, count)
	}

	dataBytes := map[string]int{
		"SelectBytesScanned":  1100000000,
		"SelectBytesReturned": 1200000000,
	}

	for metric, bytes := range dataBytes {
		stubDataBytes(stub, metric, bytes)
	}

	args := resources.S3Bucket{
		Name:   "test-bucket",
		Region: "us-east-1",
	}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)

	assert.Equal(t, map[string]interface{}{
		"standard": map[string]interface{}{
			"storage_gb":                      float64(0),
			"monthly_tier_1_requests":         int64(600),
			"monthly_tier_2_requests":         int64(1500),
			"monthly_select_data_scanned_gb":  1.1,
			"monthly_select_data_returned_gb": 1.2,
		},
		"intelligent_tiering": map[string]interface{}{
			"frequent_access_storage_gb": 2.2,
		},
	}, estimates.usage)
}
