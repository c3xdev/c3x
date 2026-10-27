package aws

import (
	"context"
	"errors"

	"github.com/c3xdev/c3x/internal/usagesync"
)

const (
	kindS3Bucket         = "aws_s3_bucket"
	keyStandardStorageGB = "standard_storage_gb"

	// bytesPerGB: S3 prices per GB where 1 GB is 2^30 bytes.
	bytesPerGB = 1 << 30

	// BucketSizeBytes is published daily.
	secondsPerDay = 24 * 60 * 60
)

// s3Bucket measures storage and request counts independently. Request
// metrics exist only where a request-metrics configuration is present, so
// they fail for most buckets; that failure must not drop storage, which
// every bucket publishes.
func (s *Source) s3Bucket(ctx context.Context, t usagesync.Target, w usagesync.Window) (usagesync.Result, error) {
	storage, storageErr := s.s3Storage(ctx, t, w)
	requests, requestsErr := s.s3Requests(ctx, t, w)
	return mergeParts(
		collected{res: storage, err: storageErr},
		collected{res: requests, err: requestsErr},
	)
}

// s3Storage reads the bucket's Standard storage size. CloudWatch publishes
// BucketSizeBytes for every bucket once a day, with no configuration.
// standard_storage_gb is the mean of the daily sizes in the window, the
// quantity a GB-month price applies to. series holds the same mean per
// calendar month (UTC).
func (s *Source) s3Storage(ctx context.Context, t usagesync.Target, w usagesync.Window) (usagesync.Result, error) {
	points, err := datapoints(ctx, s.client(t.Region), metricRequest{
		namespace: "AWS/S3",
		metric:    "BucketSizeBytes",
		stat:      "Average",
		dimensions: map[string]string{
			"BucketName":  t.ID,
			"StorageType": "StandardStorage",
		},
		periodSecs: secondsPerDay,
		start:      w.Start,
		end:        w.End,
	})
	if err != nil {
		return usagesync.Result{}, err
	}
	if len(points) == 0 {
		return usagesync.Result{}, errors.New("no BucketSizeBytes datapoints in the window")
	}

	var sum float64
	monthSum := map[string]float64{}
	monthN := map[string]int{}
	for _, p := range points {
		sum += p.value
		m := p.at.UTC().Format("2006-01")
		monthSum[m] += p.value
		monthN[m]++
	}
	series := make(map[string]float64, len(monthSum))
	for m, total := range monthSum {
		series[m] = toGB(total / float64(monthN[m]))
	}
	return usagesync.Result{
		Values: map[string]float64{keyStandardStorageGB: toGB(sum / float64(len(points)))},
		Series: map[string]map[string]float64{keyStandardStorageGB: series},
	}, nil
}

// toGB converts bytes to GB, rounded to four decimals so a tiny bucket is
// not reported as zero and a large one is not cluttered with noise.
func toGB(bytes float64) float64 {
	return round4(bytes / bytesPerGB)
}
