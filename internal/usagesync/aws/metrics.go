package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// maxPages bounds pagination so a misbehaving endpoint cannot loop forever.
const maxPages = 50

type datapoint struct {
	at    time.Time
	value float64
}

type metricRequest struct {
	namespace  string
	metric     string
	stat       string
	dimensions map[string]string
	periodSecs int32
	start, end time.Time
}

// datapoints reads one metric with GetMetricData, following NextToken.
func datapoints(ctx context.Context, cw CloudWatch, r metricRequest) ([]datapoint, error) {
	dims := make([]types.Dimension, 0, len(r.dimensions))
	for name, value := range r.dimensions {
		dims = append(dims, types.Dimension{Name: ptr(name), Value: ptr(value)})
	}
	in := &cloudwatch.GetMetricDataInput{
		StartTime: &r.start,
		EndTime:   &r.end,
		ScanBy:    types.ScanByTimestampAscending,
		MetricDataQueries: []types.MetricDataQuery{{
			Id:         ptr("m0"),
			ReturnData: ptr(true),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  ptr(r.namespace),
					MetricName: ptr(r.metric),
					Dimensions: dims,
				},
				Period: ptr(r.periodSecs),
				Stat:   ptr(r.stat),
			},
		}},
	}

	var out []datapoint
	for page := 0; page < maxPages; page++ {
		resp, err := cw.GetMetricData(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("CloudWatch GetMetricData %s/%s: %w", r.namespace, r.metric, err)
		}
		for _, res := range resp.MetricDataResults {
			n := min(len(res.Timestamps), len(res.Values))
			for i := 0; i < n; i++ {
				out = append(out, datapoint{at: res.Timestamps[i], value: res.Values[i]})
			}
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			return out, nil
		}
		in.NextToken = resp.NextToken
	}
	return nil, fmt.Errorf("CloudWatch GetMetricData %s/%s: more than %d pages", r.namespace, r.metric, maxPages)
}

func ptr[T any](v T) *T { return &v }
