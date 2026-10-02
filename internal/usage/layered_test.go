package usage_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/usage"
)

func loadBoth(t *testing.T, syncedYAML, handYAML string) (synced, hand usage.File) {
	t.Helper()
	var err error
	if synced, err = usage.Load(write(t, t.TempDir(), syncedYAML)); err != nil {
		t.Fatalf("synced: %v", err)
	}
	if hand, err = usage.Load(write(t, t.TempDir(), handYAML)); err != nil {
		t.Fatalf("hand: %v", err)
	}
	return synced, hand
}

func bucket(name string) domain.Resource {
	return domain.Resource{Ref: domain.Reference{Kind: "aws_s3_bucket", Name: name}}
}

func TestApplyLayered_HandWinsPerKeySyncedFillsTheRest(t *testing.T) {
	t.Parallel()
	synced, hand := loadBoth(t, `
version: "0.1"
resource_usage:
  aws_s3_bucket.data:
    standard_storage_gb: 500
    monthly_tier_1_requests: 7000
  aws_s3_bucket.logs:
    standard_storage_gb: 20
`, `
version: "0.1"
resource_usage:
  aws_s3_bucket.data:
    standard_storage_gb: 10
`)
	resources := []domain.Resource{bucket("data"), bucket("logs")}
	rep := usage.ApplyLayered(resources, synced, hand)

	if got := resources[0].Attributes["standard_storage_gb"]; got != 10 {
		t.Errorf("data storage = %v, want the hand-written 10 to win", got)
	}
	if got := resources[0].Attributes["monthly_tier_1_requests"]; got != 7000 {
		t.Errorf("data requests = %v, want the synced 7000 for the key the hand file omits", got)
	}
	if got := resources[1].Attributes["standard_storage_gb"]; got != 20 {
		t.Errorf("logs storage = %v, want the synced 20 for an address the hand file omits", got)
	}
	if len(rep.Unmatched) != 0 || len(rep.UnmatchedSynced) != 0 {
		t.Errorf("unexpected unmatched: %+v", rep)
	}
}

func TestApplyLayered_PrecedenceOrder(t *testing.T) {
	t.Parallel()
	// Lowest to highest: synced defaults, hand defaults, synced
	// resource_usage, hand resource_usage. Each key below is set by a
	// different subset of layers, so the winner shows which layer won.
	synced, hand := loadBoth(t, `
version: "0.1"
defaults:
  aws_s3_bucket: {a: synced-default, b: synced-default, c: synced-default, d: synced-default}
resource_usage:
  aws_s3_bucket.data: {c: synced-address, d: synced-address}
`, `
version: "0.1"
defaults:
  aws_s3_bucket: {b: hand-default, c: hand-default, d: hand-default}
resource_usage:
  aws_s3_bucket.data: {d: hand-address}
`)
	resources := []domain.Resource{bucket("data")}
	usage.ApplyLayered(resources, synced, hand)

	want := map[string]any{
		"a": "synced-default", // only the lowest layer sets it
		"b": "hand-default",   // hand default beats synced default
		"c": "synced-address", // a per-address measurement beats a kind-wide hand default
		"d": "hand-address",   // a hand entry for the exact address beats everything
	}
	if !reflect.DeepEqual(resources[0].Attributes, want) {
		t.Errorf("attributes = %v\nwant %v", resources[0].Attributes, want)
	}
}

func TestApplyLayered_SyncedOnlyOrHandOnlyBehavesLikeApply(t *testing.T) {
	t.Parallel()
	f, _ := loadBoth(t, `
version: "0.1"
resource_usage:
  aws_s3_bucket.data: {standard_storage_gb: 5}
`, `version: "0.1"`)

	viaApply := []domain.Resource{bucket("data")}
	usage.Apply(viaApply, f)
	viaLayered := []domain.Resource{bucket("data")}
	usage.ApplyLayered(viaLayered, f, usage.File{})
	if !reflect.DeepEqual(viaApply[0].Attributes, viaLayered[0].Attributes) {
		t.Errorf("Apply %v != ApplyLayered %v", viaApply[0].Attributes, viaLayered[0].Attributes)
	}
}

func TestApplyLayered_ReportsUnmatchedPerFile(t *testing.T) {
	t.Parallel()
	synced, hand := loadBoth(t, `
version: "0.1"
resource_usage:
  aws_s3_bucket.removed: {standard_storage_gb: 1}
`, `
version: "0.1"
resource_usage:
  aws_s3_bucket.typo: {standard_storage_gb: 1}
`)
	rep := usage.ApplyLayered([]domain.Resource{bucket("data")}, synced, hand)
	if !reflect.DeepEqual(rep.Unmatched, []string{"aws_s3_bucket.typo"}) {
		t.Errorf("Unmatched = %v, want the hand-written typo only", rep.Unmatched)
	}
	if !reflect.DeepEqual(rep.UnmatchedSynced, []string{"aws_s3_bucket.removed"}) {
		t.Errorf("UnmatchedSynced = %v, want the stale synced key only", rep.UnmatchedSynced)
	}
}

func TestApplyLayered_LegacyKeysInEitherFile(t *testing.T) {
	t.Parallel()
	synced, hand := loadBoth(t, `
version: "0.1"
resource_usage:
  aws_s3_bucket.module.m.data: {standard_storage_gb: 5}
`, `version: "0.1"`)
	r := domain.Resource{Ref: domain.Reference{Kind: "aws_s3_bucket", Name: "module.m.data"}}
	resources := []domain.Resource{r}
	rep := usage.ApplyLayered(resources, synced, hand)
	if resources[0].Attributes["standard_storage_gb"] != 5 {
		t.Errorf("legacy synced key not applied: %v", resources[0].Attributes)
	}
	if rep.Legacy["aws_s3_bucket.module.m.data"] != "module.m.aws_s3_bucket.data" {
		t.Errorf("Legacy = %v", rep.Legacy)
	}
}

// A synced file carries extra top-level keys; the loader must read the
// quantities and skip the rest, as the v0.3.x loaders already do.
func TestLoadIgnoresSyncedMetadata(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "c3x-usage.synced.yml")
	body := `version: "0.1"
synced:
  provider: aws
  window_days: 30
  generated_at: 2026-09-29T12:00:00Z
resource_usage:
  aws_s3_bucket.data:
    standard_storage_gb: 512.4
series:
  aws_s3_bucket.data:
    standard_storage_gb: {2026-07: 470.1, 2026-08: 498.9, 2026-09: 512.4}
errors:
  aws_s3_bucket.logs: "no BucketSizeBytes datapoints in the window"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := usage.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := f.ResourceUsage["aws_s3_bucket.data"]["standard_storage_gb"]; got != 512.4 {
		t.Errorf("standard_storage_gb = %v, want 512.4", got)
	}
}
