package aws_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/c3xdev/c3x/internal/usagesync"
)

// window30 is 30 days, so a monthly quantity is the measured total.
var window30 = usagesync.Window{Start: day(2026, 9, 1), End: day(2026, 10, 1)}

func filterMetric(bucket, filter string) types.Metric {
	return types.Metric{Dimensions: []types.Dimension{
		{Name: ptr("BucketName"), Value: ptr(bucket)},
		{Name: ptr("FilterId"), Value: ptr(filter)},
	}}
}

func listPage(next string, metrics ...types.Metric) *cloudwatch.ListMetricsOutput {
	out := &cloudwatch.ListMetricsOutput{Metrics: metrics}
	if next != "" {
		out.NextToken = &next
	}
	return out
}

func TestS3Requests_SumsTiersAndKeepsOtherRequests(t *testing.T) {
	t.Parallel()
	at := day(2026, 9, 1)
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{listPage("", filterMetric("acme-data", "logs-prefix"))},
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: at, v: 100 * gb}},
			"PutRequests":     {{at: at, v: 1000}},
			"PostRequests":    {{at: at, v: 200}},
			"ListRequests":    {{at: at, v: 50}},
			"GetRequests":     {{at: at, v: 800}},
			"SelectRequests":  {{at: at, v: 10}},
			"HeadRequests":    {{at: at, v: 20}},
			"DeleteRequests":  {{at: at, v: 5}},
			// 15 requests match none of the named metrics. Tier 2 is
			// "GET, SELECT, and other", so they are counted, not dropped.
			"AllRequests": {{at: at, v: 2100}},
		},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Problem != "" {
		t.Fatalf("problem = %q, want none", res.Problem)
	}
	if got := res.Values["standard_storage_gb"]; got != 100 {
		t.Errorf("storage = %v, want 100", got)
	}
	// tier 1 = 1000 + 200 + 50. tier 2 = 800 + 10 + 20 + 5 + 15 remainder.
	if got := res.Values["monthly_tier_1_requests"]; got != 1250 {
		t.Errorf("tier 1 = %v, want 1250", got)
	}
	if got := res.Values["monthly_tier_2_requests"]; got != 850 {
		t.Errorf("tier 2 = %v, want 850", got)
	}
	if res.Series["monthly_tier_1_requests"]["2026-09"] != 1250 || res.Series["monthly_tier_2_requests"]["2026-09"] != 850 {
		t.Errorf("series = %v", res.Series)
	}
}

func TestS3Requests_UsesTheDiscoveredFilterAndDoesNotAssumeOne(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{listPage("", filterMetric("acme-data", "logs-prefix"))},
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: day(2026, 9, 1), v: gb}},
			"PutRequests":     {{at: day(2026, 9, 1), v: 4}},
		},
	}
	var regions []string
	if _, err := source(cw, &regions).Collect(context.Background(), bucket, window30); err != nil {
		t.Fatal(err)
	}
	if len(cw.listCalls) != 1 {
		t.Fatalf("ListMetrics calls = %d, want 1", len(cw.listCalls))
	}
	in := cw.listCalls[0]
	if in.Namespace == nil || *in.Namespace != "AWS/S3" || in.MetricName == nil || *in.MetricName != "AllRequests" {
		t.Errorf("ListMetrics %v/%v, want AWS/S3 AllRequests", in.Namespace, in.MetricName)
	}
	if in.RecentlyActive != "" {
		t.Errorf("RecentlyActive = %q, want unset so a quiet bucket is still found", in.RecentlyActive)
	}
	dims := map[string]string{}
	for _, d := range in.Dimensions {
		dims[*d.Name] = *d.Value
	}
	if len(dims) != 1 || dims["BucketName"] != "acme-data" {
		t.Errorf("ListMetrics dimensions = %v", dims)
	}

	var req *cloudwatch.GetMetricDataInput
	for _, call := range cw.calls {
		if firstMetricName(call) == "PutRequests" {
			req = call
		}
	}
	if req == nil {
		t.Fatal("no request-metric call")
	}
	if !req.StartTime.Equal(window30.Start) || !req.EndTime.Equal(window30.End) {
		t.Errorf("window = %v..%v", req.StartTime, req.EndTime)
	}
	got := map[string]bool{}
	var stat string
	var period int32
	var filter string
	for _, q := range req.MetricDataQueries {
		ms := q.MetricStat
		got[*ms.Metric.MetricName] = true
		stat, period = *ms.Stat, *ms.Period
		for _, d := range ms.Metric.Dimensions {
			if *d.Name == "FilterId" {
				filter = *d.Value
			}
		}
	}
	for _, name := range []string{"PutRequests", "PostRequests", "ListRequests", "GetRequests", "SelectRequests", "HeadRequests", "DeleteRequests", "AllRequests"} {
		if !got[name] {
			t.Errorf("missing metric %s", name)
		}
	}
	if stat != "Sum" || period != 86400 {
		t.Errorf("stat = %s period = %d, want Sum over a day", stat, period)
	}
	if filter != "logs-prefix" {
		t.Errorf("FilterId = %q, want the discovered configuration, not an assumed name", filter)
	}
}

func TestS3Requests_SeriesScalesEachMonthByTheWindowInsideIt(t *testing.T) {
	t.Parallel()
	// Aug 17 to Sep 16 is 30 days: 15 in August, 15 in September.
	w := usagesync.Window{Start: day(2026, 8, 17), End: day(2026, 9, 16)}
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{listPage("", filterMetric("acme-data", "entire"))},
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: day(2026, 8, 17), v: gb}},
			"PutRequests":     {{at: day(2026, 8, 17), v: 100}, {at: day(2026, 9, 1), v: 50}},
		},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, w)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Values["monthly_tier_1_requests"]; got != 150 {
		t.Errorf("value = %v, want the 30-day total 150", got)
	}
	got := res.Series["monthly_tier_1_requests"]
	// 100 requests over August's 15 covered days, expressed per 30 days.
	if got["2026-08"] != 200 || got["2026-09"] != 100 || len(got) != 2 {
		t.Errorf("series = %v, want {2026-08: 200, 2026-09: 100}", got)
	}
}

func TestS3Requests_NoConfigurationKeepsStorage(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{valuesByMetric: map[string][]cwPoint{
		"BucketSizeBytes": {{at: day(2026, 9, 1), v: 100 * gb}},
	}}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatalf("err = %v, want storage kept", err)
	}
	if res.Values["standard_storage_gb"] != 100 {
		t.Errorf("storage = %v, want 100", res.Values["standard_storage_gb"])
	}
	if _, ok := res.Values["monthly_tier_1_requests"]; ok {
		t.Errorf("tier 1 = %v, want the key unset rather than guessed as zero", res.Values["monthly_tier_1_requests"])
	}
	if !strings.Contains(res.Problem, "no S3 request metrics") {
		t.Errorf("problem = %q", res.Problem)
	}
	if res.Series["standard_storage_gb"] == nil {
		t.Error("storage series was dropped with the request failure")
	}
}

func TestS3Requests_SeveralConfigurationsAreNotAdded(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{
			listPage("next", filterMetric("acme-data", "zeta")),
			listPage("", filterMetric("acme-data", "alpha")),
		},
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: day(2026, 9, 1), v: gb}},
			"PutRequests":     {{at: day(2026, 9, 1), v: 999}},
		},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Values["monthly_tier_1_requests"]; ok {
		t.Error("request counts were written; overlapping filters must not be summed")
	}
	if res.Values["standard_storage_gb"] != 1 {
		t.Errorf("storage = %v, want it kept", res.Values["standard_storage_gb"])
	}
	if !strings.Contains(res.Problem, "alpha") || !strings.Contains(res.Problem, "zeta") || !strings.Contains(res.Problem, "not adding them together") {
		t.Errorf("problem = %q", res.Problem)
	}
	// The second page has to be read. Stopping at the first filter would
	// silently measure one prefix and miss that another exists.
	if len(cw.listCalls) != 2 || cw.listCalls[1].NextToken == nil || *cw.listCalls[1].NextToken != "next" {
		t.Errorf("list pagination = %+v", cw.listCalls)
	}
	for _, call := range cw.calls {
		if firstMetricName(call) == "PutRequests" {
			t.Error("request metrics were queried after the configurations were refused")
		}
	}
}

func TestS3Requests_RepeatedFilterIsOneConfiguration(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{
			listPage("next", filterMetric("acme-data", "entire")),
			listPage("", filterMetric("acme-data", "entire")),
		},
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: day(2026, 9, 1), v: gb}},
			"PutRequests":     {{at: day(2026, 9, 1), v: 6}},
		},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Problem != "" || res.Values["monthly_tier_1_requests"] != 6 {
		t.Errorf("problem = %q values = %v, want the one filter measured", res.Problem, res.Values)
	}
}

func TestS3Requests_UnexpectedDimensionShapeIsNotUsed(t *testing.T) {
	t.Parallel()
	odd := types.Metric{Dimensions: []types.Dimension{
		{Name: ptr("BucketName"), Value: ptr("acme-data")},
		{Name: ptr("FilterId"), Value: ptr("entire")},
		{Name: ptr("StorageType"), Value: ptr("StandardStorage")},
	}}
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{listPage("", odd)},
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: day(2026, 9, 1), v: gb}},
			"PutRequests":     {{at: day(2026, 9, 1), v: 999}},
		},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Values["monthly_tier_1_requests"]; ok {
		t.Error("a metric with an unexpected dimension was treated as the bucket's requests")
	}
	if !strings.Contains(res.Problem, "no S3 request metrics") {
		t.Errorf("problem = %q", res.Problem)
	}
}

func TestS3Requests_ConfiguredButQuietIsZero(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{listPage("", filterMetric("acme-data", "entire"))},
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: day(2026, 9, 1), v: gb}},
		},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Problem != "" {
		t.Fatalf("problem = %q, want a measured zero", res.Problem)
	}
	if res.Values["monthly_tier_1_requests"] != 0 || res.Values["monthly_tier_2_requests"] != 0 {
		t.Errorf("values = %v, want zeros", res.Values)
	}
	if res.Series["monthly_tier_1_requests"] != nil {
		t.Errorf("series = %v, want none for a window with no points", res.Series)
	}
}

func TestS3Requests_MetricErrorKeepsStorage(t *testing.T) {
	t.Parallel()
	denied := errors.New("AccessDenied")
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{listPage("", filterMetric("acme-data", "entire"))},
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: day(2026, 9, 1), v: 4 * gb}},
		},
		failMetric: map[string]error{"PutRequests": denied},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatalf("err = %v, want storage kept", err)
	}
	if res.Values["standard_storage_gb"] != 4 {
		t.Errorf("storage = %v", res.Values["standard_storage_gb"])
	}
	if _, ok := res.Values["monthly_tier_1_requests"]; ok {
		t.Error("request keys were written after the call failed")
	}
	if !strings.Contains(res.Problem, "CloudWatch GetMetricData") || !strings.Contains(res.Problem, "PutRequests") || !strings.Contains(res.Problem, "AccessDenied") {
		t.Errorf("problem = %q", res.Problem)
	}
}

func TestS3Requests_StorageErrorKeepsRequests(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{listPage("", filterMetric("acme-data", "entire"))},
		valuesByMetric: map[string][]cwPoint{
			"GetRequests": {{at: day(2026, 9, 1), v: 8}},
		},
		failMetric: map[string]error{"BucketSizeBytes": errors.New("throttled")},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatalf("err = %v, want the request counts kept", err)
	}
	if _, ok := res.Values["standard_storage_gb"]; ok {
		t.Error("storage was written after its call failed")
	}
	if res.Values["monthly_tier_2_requests"] != 8 {
		t.Errorf("tier 2 = %v, want 8", res.Values["monthly_tier_2_requests"])
	}
	if !strings.Contains(res.Problem, "BucketSizeBytes") {
		t.Errorf("problem = %q", res.Problem)
	}
}

func TestS3Requests_ListErrorKeepsStorage(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{
		listErr: errors.New("AccessDenied"),
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: day(2026, 9, 1), v: 2 * gb}},
		},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Values["standard_storage_gb"] != 2 {
		t.Errorf("storage = %v", res.Values["standard_storage_gb"])
	}
	if !strings.Contains(res.Problem, "CloudWatch ListMetrics AWS/S3/AllRequests") {
		t.Errorf("problem = %q", res.Problem)
	}
}

func TestS3Requests_FollowsGetMetricDataPagination(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{listPage("", filterMetric("acme-data", "entire"))},
		byName: map[string][]*cloudwatch.GetMetricDataOutput{
			"BucketSizeBytes": {page("", []time.Time{day(2026, 9, 1)}, []float64{gb})},
			"PutRequests": {
				pageResults("t1", types.MetricDataResult{Id: ptr("put"), Timestamps: []time.Time{day(2026, 9, 1)}, Values: []float64{10}}),
				pageResults("", types.MetricDataResult{Id: ptr("put"), Timestamps: []time.Time{day(2026, 9, 2)}, Values: []float64{30}}),
			},
		},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Values["monthly_tier_1_requests"] != 40 {
		t.Errorf("tier 1 = %v, want the sum across both pages, 40", res.Values["monthly_tier_1_requests"])
	}
	var sawToken bool
	for _, call := range cw.calls {
		if firstMetricName(call) == "PutRequests" && call.NextToken != nil && *call.NextToken == "t1" {
			sawToken = true
		}
	}
	if !sawToken {
		t.Error("the request-metric call did not follow NextToken")
	}
}

func TestS3Requests_NegativeGapIsNotSubtracted(t *testing.T) {
	t.Parallel()
	at := day(2026, 9, 1)
	cw := &fakeCW{
		listPages: []*cloudwatch.ListMetricsOutput{listPage("", filterMetric("acme-data", "entire"))},
		valuesByMetric: map[string][]cwPoint{
			"BucketSizeBytes": {{at: at, v: gb}},
			"PutRequests":     {{at: at, v: 50}},
			"AllRequests":     {{at: at, v: 30}},
		},
	}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window30)
	if err != nil {
		t.Fatal(err)
	}
	if res.Values["monthly_tier_1_requests"] != 50 {
		t.Errorf("tier 1 = %v, want 50; a short AllRequests must not shrink the named metrics", res.Values["monthly_tier_1_requests"])
	}
	if res.Values["monthly_tier_2_requests"] != 0 {
		t.Errorf("tier 2 = %v, want 0", res.Values["monthly_tier_2_requests"])
	}
}

func pageResults(next string, results ...types.MetricDataResult) *cloudwatch.GetMetricDataOutput {
	out := &cloudwatch.GetMetricDataOutput{MetricDataResults: results}
	if next != "" {
		out.NextToken = &next
	}
	return out
}
