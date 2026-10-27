package aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/c3xdev/c3x/internal/usagesync"
)

const (
	keyMonthlyTier1Requests = "monthly_tier_1_requests"
	keyMonthlyTier2Requests = "monthly_tier_2_requests"
)

// s3RequestMetric is one CloudWatch request metric and the catalog tier
// it belongs to. tier 0 is AllRequests, which is not itself a tier: a
// positive gap between it and the named metrics is "other" and is added
// to tier 2, because the catalog's tier 2 line is GET, SELECT and the rest.
//
// PutRequests counts CopyObject at the destination, which is tier 1.
// GetRequests includes non-billable CopyObject and replication GETs;
// AWS documents that, and those requests cannot be told apart from
// billable GETs, so they are not subtracted.
var s3RequestMetrics = []struct {
	id   string
	name string
	tier int
}{
	{"put", "PutRequests", 1},
	{"post", "PostRequests", 1},
	{"list", "ListRequests", 1},
	{"get", "GetRequests", 2},
	{"select", "SelectRequests", 2},
	{"head", "HeadRequests", 2},
	{"delete", "DeleteRequests", 2},
	{"all", "AllRequests", 0},
}

// s3Requests reads monthly_tier_1_requests and monthly_tier_2_requests.
// Request metrics are published only for a bucket with a request-metrics
// configuration, which is opt-in and billed, and only with that
// configuration's FilterId. The id is discovered, never assumed: none
// means the counts are unknown, and more than one is not summed because
// filters can overlap or cover only a prefix.
func (s *Source) s3Requests(ctx context.Context, t usagesync.Target, w usagesync.Window) (usagesync.Result, error) {
	listed, err := listMetrics(ctx, s.client(t.Region), "AWS/S3", "AllRequests", map[string]string{
		"BucketName": t.ID,
	})
	if err != nil {
		return usagesync.Result{}, err
	}
	filter, err := s3RequestFilter(listed, t.ID)
	if err != nil {
		return usagesync.Result{}, err
	}

	metrics := make([]namedMetric, len(s3RequestMetrics))
	for i, m := range s3RequestMetrics {
		metrics[i] = namedMetric{id: m.id, name: m.name}
	}
	series, err := metricSeries(ctx, s.client(t.Region), seriesRequest{
		namespace: "AWS/S3",
		stat:      "Sum",
		dimensions: map[string]string{
			"BucketName": t.ID,
			"FilterId":   filter,
		},
		periodSecs: secondsPerDay,
		start:      w.Start,
		end:        w.End,
		metrics:    metrics,
	})
	if err != nil {
		return usagesync.Result{}, err
	}

	var tier1, tier2, explicit []datapoint
	for _, m := range s3RequestMetrics {
		switch m.tier {
		case 1:
			tier1 = append(tier1, series[m.name]...)
		case 2:
			tier2 = append(tier2, series[m.name]...)
		}
		if m.tier != 0 {
			explicit = append(explicit, series[m.name]...)
		}
	}
	tier2 = append(tier2, positiveRemainder(series["AllRequests"], explicit)...)

	v1, s1 := monthlyCount(tier1, w)
	v2, s2 := monthlyCount(tier2, w)
	res := usagesync.Result{Values: map[string]float64{
		keyMonthlyTier1Requests: v1,
		keyMonthlyTier2Requests: v2,
	}}
	if len(s1) > 0 || len(s2) > 0 {
		res.Series = map[string]map[string]float64{}
		if len(s1) > 0 {
			res.Series[keyMonthlyTier1Requests] = s1
		}
		if len(s2) > 0 {
			res.Series[keyMonthlyTier2Requests] = s2
		}
	}
	return res, nil
}

// s3RequestFilter is the one FilterId to query. A metric whose dimensions
// are not exactly BucketName and FilterId is ignored: a different shape
// is not guessed at.
func s3RequestFilter(metrics []map[string]string, bucket string) (string, error) {
	seen := map[string]bool{}
	var ids []string
	for _, dims := range metrics {
		if dims["BucketName"] != bucket || len(dims) != 2 {
			continue
		}
		id := dims["FilterId"]
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	sort.Strings(ids)
	switch len(ids) {
	case 0:
		return "", errors.New("no S3 request metrics for this bucket; CloudWatch publishes them only for a bucket with a request-metrics configuration, which is opt-in")
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("multiple S3 request-metrics configurations (%s); not adding them together", strings.Join(ids, ", "))
	}
}

// positiveRemainder is AllRequests minus the named metrics, per timestamp,
// kept only when it is above zero. A negative gap is the metrics
// disagreeing (they are best-effort) and is not used to shrink a count.
func positiveRemainder(all, explicit []datapoint) []datapoint {
	total := sumByTime(all)
	if len(total) == 0 {
		return nil
	}
	known := sumByTime(explicit)
	var out []datapoint
	for at, v := range total {
		if d := v - known[at]; d > 0 {
			out = append(out, datapoint{at: at, value: d})
		}
	}
	return out
}

func sumByTime(points []datapoint) map[time.Time]float64 {
	out := map[time.Time]float64{}
	for _, p := range points {
		out[p.at] += p.value
	}
	return out
}
