package aws

import (
	"context"
	"fmt"
	"strings"
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

// namedMetric is one series inside a GetMetricData call. id is the query
// id CloudWatch echoes back (it must start with a lowercase letter);
// name is the metric.
type namedMetric struct {
	id, name string
}

// seriesRequest reads several metrics that share a namespace, stat,
// period, dimension set and window. One call, so a bucket's request
// metrics are not eight round trips; AWS still bills each metric.
type seriesRequest struct {
	namespace  string
	stat       string
	dimensions map[string]string
	periodSecs int32
	start, end time.Time
	metrics    []namedMetric
}

// datapoints reads one metric with GetMetricData, following NextToken.
func datapoints(ctx context.Context, cw CloudWatch, r metricRequest) ([]datapoint, error) {
	got, err := metricSeries(ctx, cw, seriesRequest{
		namespace:  r.namespace,
		stat:       r.stat,
		dimensions: r.dimensions,
		periodSecs: r.periodSecs,
		start:      r.start,
		end:        r.end,
		metrics:    []namedMetric{{id: "m0", name: r.metric}},
	})
	if err != nil {
		return nil, err
	}
	return got[r.metric], nil
}

// metricSeries reads every named metric. The result is keyed by metric
// name. A metric with no datapoints is absent, which the caller reads as
// zero rather than as a failure: CloudWatch omits a count it did not see.
func metricSeries(ctx context.Context, cw CloudWatch, r seriesRequest) (map[string][]datapoint, error) {
	dims := make([]types.Dimension, 0, len(r.dimensions))
	for name, value := range r.dimensions {
		dims = append(dims, types.Dimension{Name: ptr(name), Value: ptr(value)})
	}
	idToName := make(map[string]string, len(r.metrics))
	queries := make([]types.MetricDataQuery, 0, len(r.metrics))
	names := make([]string, 0, len(r.metrics))
	for _, m := range r.metrics {
		idToName[m.id] = m.name
		names = append(names, m.name)
		queries = append(queries, types.MetricDataQuery{
			Id:         ptr(m.id),
			ReturnData: ptr(true),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  ptr(r.namespace),
					MetricName: ptr(m.name),
					Dimensions: dims,
				},
				Period: ptr(r.periodSecs),
				Stat:   ptr(r.stat),
			},
		})
	}
	in := &cloudwatch.GetMetricDataInput{
		StartTime:         &r.start,
		EndTime:           &r.end,
		ScanBy:            types.ScanByTimestampAscending,
		MetricDataQueries: queries,
	}
	label := r.namespace + "/" + strings.Join(names, "+")

	out := map[string][]datapoint{}
	for page := 0; page < maxPages; page++ {
		resp, err := cw.GetMetricData(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("CloudWatch GetMetricData %s: %w", label, err)
		}
		if resp == nil {
			return nil, fmt.Errorf("CloudWatch GetMetricData %s: empty response", label)
		}
		for _, res := range resp.MetricDataResults {
			name := idToName[deref(res.Id)]
			if name == "" {
				continue
			}
			n := min(len(res.Timestamps), len(res.Values))
			for i := 0; i < n; i++ {
				out[name] = append(out[name], datapoint{at: res.Timestamps[i], value: res.Values[i]})
			}
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			return out, nil
		}
		in.NextToken = resp.NextToken
	}
	return nil, fmt.Errorf("CloudWatch GetMetricData %s: more than %d pages", label, maxPages)
}

// listMetrics returns each matching metric's dimensions. RecentlyActive is
// left unset: that flag only returns metrics touched in the last three
// hours, which would hide a bucket that was quiet today.
func listMetrics(ctx context.Context, cw CloudWatch, namespace, metric string, dimensions map[string]string) ([]map[string]string, error) {
	filters := make([]types.DimensionFilter, 0, len(dimensions))
	for name, value := range dimensions {
		filters = append(filters, types.DimensionFilter{Name: ptr(name), Value: ptr(value)})
	}
	in := &cloudwatch.ListMetricsInput{
		Namespace:  ptr(namespace),
		MetricName: ptr(metric),
		Dimensions: filters,
	}
	var out []map[string]string
	for page := 0; page < maxPages; page++ {
		resp, err := cw.ListMetrics(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("CloudWatch ListMetrics %s/%s: %w", namespace, metric, err)
		}
		if resp == nil {
			return nil, fmt.Errorf("CloudWatch ListMetrics %s/%s: empty response", namespace, metric)
		}
		for _, m := range resp.Metrics {
			dims := make(map[string]string, len(m.Dimensions))
			for _, d := range m.Dimensions {
				if d.Name == nil || d.Value == nil {
					continue
				}
				dims[*d.Name] = *d.Value
			}
			out = append(out, dims)
		}
		if resp.NextToken == nil || *resp.NextToken == "" {
			return out, nil
		}
		in.NextToken = resp.NextToken
	}
	return nil, fmt.Errorf("CloudWatch ListMetrics %s/%s: more than %d pages", namespace, metric, maxPages)
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ptr[T any](v T) *T { return &v }
