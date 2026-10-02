package usagesync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/usage"
	"github.com/c3xdev/c3x/internal/usagesync"
)

// fakeSource answers from a table and records what it was asked.
type fakeSource struct {
	results map[string]usagesync.Result
	errs    map[string]error
	panics  map[string]bool
	calls   atomic.Int32
	mu      sync.Mutex
	window  usagesync.Window
}

func (f *fakeSource) Provider() string { return "fake" }
func (f *fakeSource) Emits() map[string][]string {
	return map[string][]string{"aws_s3_bucket": {"standard_storage_gb"}}
}

func (f *fakeSource) Collect(_ context.Context, t usagesync.Target, w usagesync.Window) (usagesync.Result, error) {
	f.calls.Add(1)
	f.mu.Lock()
	f.window = w
	f.mu.Unlock()
	if f.panics[t.Address] {
		panic("boom")
	}
	if err := f.errs[t.Address]; err != nil {
		return usagesync.Result{}, err
	}
	return f.results[t.Address], nil
}

func s3(name, bucket string, region *string) domain.Resource {
	attrs := map[string]any{}
	if bucket != "" {
		attrs["bucket"] = bucket
	}
	return domain.Resource{
		Ref:        domain.Reference{Kind: "aws_s3_bucket", Name: name},
		Attributes: attrs,
		Region:     region,
	}
}

func ptr(s string) *string { return &s }

var emits = map[string][]string{"aws_s3_bucket": {"standard_storage_gb"}}

func TestResolve_UsesConfigurationLiteralWithoutState(t *testing.T) {
	t.Parallel()
	targets, problems := usagesync.Resolve(
		[]domain.Resource{s3("data", "acme-data", ptr("eu-west-1"))}, nil, false, "", emits)
	want := []usagesync.Target{{Address: "aws_s3_bucket.data", Kind: "aws_s3_bucket", ID: "acme-data", Region: "eu-west-1"}}
	if !reflect.DeepEqual(targets, want) || len(problems) != 0 {
		t.Errorf("targets = %+v, problems = %v", targets, problems)
	}
}

func TestResolve_StateIsTheSourceOfTruth(t *testing.T) {
	t.Parallel()
	// Configuration says one name and region, state says what was created.
	cfg := []domain.Resource{s3("data", "acme-data", ptr("us-east-1"))}
	state := []domain.Resource{s3("data", "acme-data-7f3a", ptr("ap-northeast-1"))}
	targets, problems := usagesync.Resolve(cfg, state, true, "", emits)
	if len(problems) != 0 || len(targets) != 1 {
		t.Fatalf("targets = %+v, problems = %v", targets, problems)
	}
	if targets[0].ID != "acme-data-7f3a" || targets[0].Region != "ap-northeast-1" {
		t.Errorf("target = %+v, want the state's name and region", targets[0])
	}
}

func TestResolve_RegionFallsBackFromStateToConfigurationToFlag(t *testing.T) {
	t.Parallel()
	cfg := []domain.Resource{s3("a", "a", ptr("eu-west-1")), s3("b", "b", nil)}
	state := []domain.Resource{s3("a", "a", nil), s3("b", "b", nil)}
	targets, problems := usagesync.Resolve(cfg, state, true, "us-west-2", emits)
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	got := map[string]string{targets[0].Address: targets[0].Region, targets[1].Address: targets[1].Region}
	want := map[string]string{"aws_s3_bucket.a": "eu-west-1", "aws_s3_bucket.b": "us-west-2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("regions = %v, want %v", got, want)
	}
}

func TestResolve_NeverGuesses(t *testing.T) {
	t.Parallel()
	cfg := []domain.Resource{
		s3("computed", "", ptr("us-east-1")),   // name not a literal
		s3("noregion", "x", nil),               // region unknown
		s3("unapplied", "y", ptr("us-east-1")), // absent from the state
		{Ref: domain.Reference{Kind: "aws_instance", Name: "web"}},
	}
	_, problems := usagesync.Resolve(cfg[:2], nil, false, "", emits)
	if !strings.Contains(problems["aws_s3_bucket.computed"], "pass --state") {
		t.Errorf("computed: %q", problems["aws_s3_bucket.computed"])
	}
	if !strings.Contains(problems["aws_s3_bucket.noregion"], "pass --region") {
		t.Errorf("noregion: %q", problems["aws_s3_bucket.noregion"])
	}

	targets, problems := usagesync.Resolve(cfg, []domain.Resource{s3("computed", "real", nil)}, true, "", emits)
	if !strings.Contains(problems["aws_s3_bucket.unapplied"], "not in the Terraform state") {
		t.Errorf("unapplied: %q", problems["aws_s3_bucket.unapplied"])
	}
	// With state, "computed" resolves: the name comes from state, the
	// region from configuration. The other two stay unresolved.
	if len(targets) != 1 || targets[0].ID != "real" || targets[0].Region != "us-east-1" {
		t.Errorf("targets = %+v, want only the computed bucket, resolved from state", targets)
	}
	if _, ok := problems["aws_instance.web"]; ok {
		t.Error("a kind the source does not support must be ignored, not reported")
	}
}

func TestRun_FailureOfOneResourceDoesNotStopTheRest(t *testing.T) {
	t.Parallel()
	src := &fakeSource{
		results: map[string]usagesync.Result{
			"aws_s3_bucket.ok": {
				Values: map[string]float64{"standard_storage_gb": 12.5},
				Series: map[string]map[string]float64{"standard_storage_gb": {"2026-09": 12.5}},
			},
		},
		errs:   map[string]error{"aws_s3_bucket.denied": errors.New("AccessDenied")},
		panics: map[string]bool{"aws_s3_bucket.bad": true},
	}
	targets := []usagesync.Target{
		{Address: "aws_s3_bucket.ok"}, {Address: "aws_s3_bucket.denied"}, {Address: "aws_s3_bucket.bad"},
	}
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	snap, err := usagesync.Run(context.Background(), usagesync.Options{
		Source: src, Targets: targets, WindowDays: 30, Now: now,
		Problems: map[string]string{"aws_s3_bucket.unresolved": "region unknown; pass --region"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := snap.ResourceUsage["aws_s3_bucket.ok"]["standard_storage_gb"]; got != 12.5 {
		t.Errorf("ok = %v, want 12.5", got)
	}
	for _, addr := range []string{"aws_s3_bucket.denied", "aws_s3_bucket.bad", "aws_s3_bucket.unresolved"} {
		if _, ok := snap.ResourceUsage[addr]; ok {
			t.Errorf("%s has usage; a failed resource must have none", addr)
		}
		if snap.Errors[addr] == "" {
			t.Errorf("%s has no error recorded", addr)
		}
	}
	if snap.Errors["aws_s3_bucket.denied"] != "AccessDenied" {
		t.Errorf("denied = %q", snap.Errors["aws_s3_bucket.denied"])
	}
	if snap.Series["aws_s3_bucket.ok"]["standard_storage_gb"]["2026-09"] != 12.5 {
		t.Errorf("series = %v", snap.Series)
	}

	wantStart := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	if !src.window.Start.Equal(wantStart) || !src.window.End.Equal(now) {
		t.Errorf("window = %v..%v, want %v..%v", src.window.Start, src.window.End, wantStart, now)
	}
	if snap.Synced.Provider != "fake" || snap.Synced.WindowDays != 30 || !snap.Synced.GeneratedAt.Equal(now) {
		t.Errorf("synced = %+v", snap.Synced)
	}
}

func TestRun_StopsWhenTheContextEnds(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	src := &fakeSource{}
	_, err := usagesync.Run(ctx, usagesync.Options{
		Source: src, Targets: []usagesync.Target{{Address: "a"}, {Address: "b"}}, WindowDays: 30,
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

// What sync writes must load with the existing loader and price from
// resource_usage, ignoring the metadata.
func TestSnapshotRoundTripsThroughTheUsageLoader(t *testing.T) {
	t.Parallel()
	src := &fakeSource{results: map[string]usagesync.Result{
		"aws_s3_bucket.data": {
			Values: map[string]float64{"standard_storage_gb": 512.4},
			Series: map[string]map[string]float64{"standard_storage_gb": {"2026-08": 498.9, "2026-09": 512.4}},
		},
	}}
	snap, err := usagesync.Run(context.Background(), usagesync.Options{
		Source: src, Targets: []usagesync.Target{{Address: "aws_s3_bucket.data"}}, WindowDays: 30,
		Problems: map[string]string{"aws_s3_bucket.logs": "no BucketSizeBytes datapoints in the window"},
		Now:      time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "c3x-usage.synced.yml")
	if err := snap.Write(path); err != nil {
		t.Fatal(err)
	}

	f, err := usage.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := f.ResourceUsage["aws_s3_bucket.data"]["standard_storage_gb"]; got != 512.4 {
		t.Errorf("standard_storage_gb = %v, want 512.4", got)
	}

	raw, _ := os.ReadFile(path)
	text := string(raw)
	for _, want := range []string{"# Generated by `c3x usage sync`", "synced:", "series:", "errors:", "no BucketSizeBytes datapoints"} {
		if !strings.Contains(text, want) {
			t.Errorf("file lacks %q:\n%s", want, text)
		}
	}
}

func TestWriteReplacesTheFileWholeAndLeavesNoTemporaries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "c3x-usage.synced.yml")
	if err := os.WriteFile(path, []byte("stale: content that must disappear\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snap := usagesync.Snapshot{
		Version:       usagesync.SchemaVersion,
		ResourceUsage: map[string]map[string]any{"aws_s3_bucket.data": {"standard_storage_gb": 1.5}},
	}
	if err := snap.Write(path); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "stale") {
		t.Errorf("the previous contents survived:\n%s", raw)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("directory has %d entries, want only the snapshot", len(entries))
	}
}

func TestWriteFailureLeavesThePreviousFile(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "no-such-dir", "c3x-usage.synced.yml")
	if err := (usagesync.Snapshot{Version: usagesync.SchemaVersion}).Write(missing); err == nil {
		t.Fatal("want an error when the directory does not exist")
	}
}
