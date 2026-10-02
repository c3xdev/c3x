package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeBucketProject makes the current directory a project with one S3
// bucket and returns it. The project config is read from the working
// directory, so the tests below change into it.
func writeBucketProject(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	tf := `
		provider "aws" { region = "us-east-1" }
		resource "aws_s3_bucket" "data" { bucket = "acme-data" }
	`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(tf), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	return dir
}

func writeProjectFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// bucketSubtotal estimates the project in the working directory and
// returns the bucket's monthly subtotal.
func bucketSubtotal(t *testing.T) string {
	t.Helper()
	out, err := runCLI(t, "estimate", "--path", ".", "--format", "json",
		"--pricing-endpoint", flatPricing(t), "--no-cache")
	if err != nil {
		t.Fatalf("estimate: %v\n%s", err, out)
	}
	start := strings.Index(out, "{")
	if start < 0 {
		t.Fatalf("no JSON in output:\n%s", out)
	}
	var got struct {
		Costs []struct {
			Resource string `json:"resource"`
			Subtotal string `json:"monthly_subtotal"`
		} `json:"costs"`
	}
	if err := json.NewDecoder(strings.NewReader(out[start:])).Decode(&got); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	for _, c := range got.Costs {
		if c.Resource == "aws_s3_bucket.data" {
			return c.Subtotal
		}
	}
	t.Fatalf("aws_s3_bucket.data not in the estimate:\n%s", out)
	return ""
}

const syncedBucket = `version: "0.1"
synced:
  provider: aws
  window_days: 30
resource_usage:
  aws_s3_bucket.data:
    standard_storage_gb: 500
series:
  aws_s3_bucket.data:
    standard_storage_gb: {2026-08: 480, 2026-09: 500}
errors:
  aws_s3_bucket.logs: "no BucketSizeBytes datapoints in the window"
`

// The snapshot next to the project is picked up with no configuration,
// and it changes the usage-driven line.
func TestSyncedUsageFileChangesUsageLines(t *testing.T) {
	dir := writeBucketProject(t)
	without := bucketSubtotal(t)

	writeProjectFile(t, dir, "c3x-usage.synced.yml", syncedBucket)
	with := bucketSubtotal(t)

	if with == without {
		t.Errorf("subtotal %s did not change once a synced file supplied storage", with)
	}
}

// The hand-written file wins per key, and a synced value still fills
// every key the hand-written file leaves out.
func TestHandWrittenUsageWinsOverSynced(t *testing.T) {
	dir := writeBucketProject(t)
	writeProjectFile(t, dir, ".c3x.toml", `usage_path = "c3x-usage.yml"`+"\n")
	writeProjectFile(t, dir, "c3x-usage.yml", `version: "0.1"`+"\n")
	writeProjectFile(t, dir, "c3x-usage.synced.yml", syncedBucket)
	syncedOnly := bucketSubtotal(t)

	writeProjectFile(t, dir, "c3x-usage.yml", `version: "0.1"
resource_usage:
  aws_s3_bucket.data:
    standard_storage_gb: 10
`)
	both := bucketSubtotal(t)
	if both == syncedOnly {
		t.Fatalf("hand-written storage did not override the synced %s", syncedOnly)
	}

	if err := os.Remove(filepath.Join(dir, "c3x-usage.synced.yml")); err != nil {
		t.Fatal(err)
	}
	handOnly := bucketSubtotal(t)
	if both != handOnly {
		t.Errorf("hand + synced = %s, hand only = %s; the hand-written 10 should decide storage", both, handOnly)
	}

	// A key only the synced file sets still applies underneath.
	writeProjectFile(t, dir, "c3x-usage.synced.yml", syncedBucket)
	writeProjectFile(t, dir, "c3x-usage.yml", `version: "0.1"
resource_usage:
  aws_s3_bucket.data:
    monthly_tier_1_requests: 1000000
`)
	mixed := bucketSubtotal(t)
	writeProjectFile(t, dir, "c3x-usage.synced.yml", `version: "0.1"`+"\n")
	requestsOnly := bucketSubtotal(t)
	if mixed == requestsOnly {
		t.Errorf("synced storage was lost when the hand-written file set only requests (%s)", mixed)
	}
}

// A snapshot that is configured but missing is an error, not silence: the
// user asked for usage that is not there.
func TestMissingConfiguredSyncedFileIsAnError(t *testing.T) {
	dir := writeBucketProject(t)
	writeProjectFile(t, dir, ".c3x.toml", `synced_usage_path = "nope.yml"`+"\n")

	out, err := runCLI(t, "estimate", "--path", ".", "--pricing-endpoint", flatPricing(t), "--no-cache")
	if err == nil || !strings.Contains(err.Error(), "synced usage file") {
		t.Fatalf("err = %v, want a synced usage file error\n%s", err, out)
	}
}
