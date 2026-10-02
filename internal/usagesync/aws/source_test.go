package aws_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/expr"
	"github.com/c3xdev/c3x/internal/usagesync"
	awssrc "github.com/c3xdev/c3x/internal/usagesync/aws"
)

const gb = 1 << 30

// fakeCW serves canned pages and records every request.
type fakeCW struct {
	pages []*cloudwatch.GetMetricDataOutput
	err   error
	calls []*cloudwatch.GetMetricDataInput
}

func (f *fakeCW) GetMetricData(_ context.Context, in *cloudwatch.GetMetricDataInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	copied := *in
	f.calls = append(f.calls, &copied)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.pages) == 0 {
		return &cloudwatch.GetMetricDataOutput{}, nil
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}

func page(next string, at []time.Time, values []float64) *cloudwatch.GetMetricDataOutput {
	out := &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{{Id: ptr("m0"), Timestamps: at, Values: values}},
	}
	if next != "" {
		out.NextToken = &next
	}
	return out
}

func ptr[T any](v T) *T { return &v }

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func source(cw awssrc.CloudWatch, regions *[]string) *awssrc.Source {
	return awssrc.NewWithClients(func(region string) awssrc.CloudWatch {
		*regions = append(*regions, region)
		return cw
	})
}

var (
	bucket = usagesync.Target{Address: "aws_s3_bucket.data", Kind: "aws_s3_bucket", ID: "acme-data", Region: "eu-west-1"}
	window = usagesync.Window{Start: day(2026, 9, 1), End: day(2026, 9, 30)}
)

func TestS3Storage_AveragesDailySizesIntoBinaryGB(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{pages: []*cloudwatch.GetMetricDataOutput{
		page("", []time.Time{day(2026, 9, 1), day(2026, 9, 2)}, []float64{100 * gb, 300 * gb}),
	}}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Values["standard_storage_gb"]; got != 200 {
		t.Errorf("standard_storage_gb = %v, want the mean 200 GB (2^30 bytes each)", got)
	}
}

func TestS3Storage_SeriesIsPerCalendarMonth(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{pages: []*cloudwatch.GetMetricDataOutput{
		page("", []time.Time{day(2026, 8, 30), day(2026, 8, 31), day(2026, 9, 1), day(2026, 9, 2)},
			[]float64{10 * gb, 20 * gb, 40 * gb, 60 * gb}),
	}}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window)
	if err != nil {
		t.Fatal(err)
	}
	got := res.Series["standard_storage_gb"]
	if got["2026-08"] != 15 || got["2026-09"] != 50 || len(got) != 2 {
		t.Errorf("series = %v, want {2026-08: 15, 2026-09: 50}", got)
	}
	if res.Values["standard_storage_gb"] != 32.5 {
		t.Errorf("value = %v, want the mean over the whole window, 32.5", res.Values["standard_storage_gb"])
	}
}

func TestS3Storage_SmallBucketIsNotRoundedToZero(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{pages: []*cloudwatch.GetMetricDataOutput{
		page("", []time.Time{day(2026, 9, 1)}, []float64{5 << 20}), // 5 MiB
	}}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Values["standard_storage_gb"]; got != 0.0049 {
		t.Errorf("value = %v, want 0.0049", got)
	}
}

func TestS3Storage_RequestShape(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{pages: []*cloudwatch.GetMetricDataOutput{page("", []time.Time{day(2026, 9, 1)}, []float64{gb})}}
	var regions []string
	if _, err := source(cw, &regions).Collect(context.Background(), bucket, window); err != nil {
		t.Fatal(err)
	}
	if len(regions) != 1 || regions[0] != "eu-west-1" {
		t.Errorf("client regions = %v, want one client for the bucket's region", regions)
	}
	if len(cw.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(cw.calls))
	}
	in := cw.calls[0]
	if !in.StartTime.Equal(window.Start) || !in.EndTime.Equal(window.End) {
		t.Errorf("window = %v..%v", in.StartTime, in.EndTime)
	}
	if len(in.MetricDataQueries) != 1 {
		t.Fatalf("queries = %d, want 1", len(in.MetricDataQueries))
	}
	ms := in.MetricDataQueries[0].MetricStat
	if *ms.Metric.Namespace != "AWS/S3" || *ms.Metric.MetricName != "BucketSizeBytes" {
		t.Errorf("metric = %s/%s", *ms.Metric.Namespace, *ms.Metric.MetricName)
	}
	if *ms.Stat != "Average" || *ms.Period != 86400 {
		t.Errorf("stat = %s period = %d, want Average over a day", *ms.Stat, *ms.Period)
	}
	dims := map[string]string{}
	for _, d := range ms.Metric.Dimensions {
		dims[*d.Name] = *d.Value
	}
	if len(dims) != 2 || dims["BucketName"] != "acme-data" || dims["StorageType"] != "StandardStorage" {
		t.Errorf("dimensions = %v", dims)
	}
}

func TestS3Storage_FollowsPagination(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{pages: []*cloudwatch.GetMetricDataOutput{
		page("t1", []time.Time{day(2026, 9, 1)}, []float64{10 * gb}),
		page("", []time.Time{day(2026, 9, 2)}, []float64{30 * gb}),
	}}
	var regions []string
	res, err := source(cw, &regions).Collect(context.Background(), bucket, window)
	if err != nil {
		t.Fatal(err)
	}
	if res.Values["standard_storage_gb"] != 20 {
		t.Errorf("value = %v, want the mean across both pages, 20", res.Values["standard_storage_gb"])
	}
	if len(cw.calls) != 2 || cw.calls[1].NextToken == nil || *cw.calls[1].NextToken != "t1" {
		t.Errorf("second call did not carry the token: %+v", cw.calls)
	}
}

func TestS3Storage_NoDatapointsIsAnExplainedError(t *testing.T) {
	t.Parallel()
	var regions []string
	_, err := source(&fakeCW{}, &regions).Collect(context.Background(), bucket, window)
	if err == nil || err.Error() != "no BucketSizeBytes datapoints in the window" {
		t.Errorf("err = %v", err)
	}
}

func TestS3Storage_APIErrorNamesTheCall(t *testing.T) {
	t.Parallel()
	denied := errors.New("AccessDenied: not authorized to perform cloudwatch:GetMetricData")
	var regions []string
	_, err := source(&fakeCW{err: denied}, &regions).Collect(context.Background(), bucket, window)
	if !errors.Is(err, denied) || !strings.Contains(err.Error(), "CloudWatch GetMetricData AWS/S3/BucketSizeBytes") {
		t.Errorf("err = %v, want the cause wrapped with the call that failed", err)
	}
}

func TestCollectRejectsAnUnsupportedKind(t *testing.T) {
	t.Parallel()
	var regions []string
	_, err := source(&fakeCW{}, &regions).Collect(context.Background(), usagesync.Target{Kind: "aws_instance"}, window)
	if err == nil || !strings.Contains(err.Error(), "does not support aws_instance") {
		t.Errorf("err = %v", err)
	}
}

func TestOneClientPerRegion(t *testing.T) {
	t.Parallel()
	cw := &fakeCW{}
	for i := 0; i < 3; i++ {
		cw.pages = append(cw.pages, page("", []time.Time{day(2026, 9, 1)}, []float64{gb}))
	}
	var regions []string
	s := source(cw, &regions)
	other := bucket
	other.Region = "us-east-1"
	for _, target := range []usagesync.Target{bucket, bucket, other} {
		if _, err := s.Collect(context.Background(), target, window); err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(regions)
	if strings.Join(regions, ",") != "eu-west-1,us-east-1" {
		t.Errorf("clients built for %v, want one per region", regions)
	}
}

// Every key the source can emit must be one the catalog reads for that
// kind. A rename in the catalog then fails here instead of syncing
// values nothing consumes.
func TestEveryEmittedKeyIsReadByTheCatalog(t *testing.T) {
	t.Parallel()
	reg, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	emits := awssrc.NewWithClients(nil).Emits()
	if len(emits) == 0 {
		t.Fatal("the source emits nothing")
	}
	for kind, keys := range emits {
		def := reg.Get(kind)
		if def == nil {
			t.Errorf("%s is not in the catalog", kind)
			continue
		}
		read := map[string]bool{}
		for _, dim := range def.Dimensions {
			for _, src := range []string{dim.Quantity, dim.When} {
				if src == "" {
					continue
				}
				ids, err := expr.Identifiers(src)
				if err != nil {
					t.Fatalf("%s: %q: %v", kind, src, err)
				}
				for _, id := range ids {
					read[id] = true
				}
			}
		}
		for _, key := range keys {
			if !read[key] {
				t.Errorf("%s: emitted key %q is not read by any catalog dimension", kind, key)
			}
		}
	}
}
