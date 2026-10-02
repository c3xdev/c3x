package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/c3xdev/c3x/internal/usagesync"
)

// fakeUsageSource stands in for the cloud: it answers from a table and
// records what it was asked, so no credentials are needed.
type fakeUsageSource struct {
	mu      sync.Mutex
	gb      map[string]float64 // bucket name → GB
	fail    map[string]error   // bucket name → error
	targets []usagesync.Target
}

func (f *fakeUsageSource) Provider() string { return "aws" }
func (f *fakeUsageSource) Emits() map[string][]string {
	return map[string][]string{"aws_s3_bucket": {"standard_storage_gb"}}
}

func (f *fakeUsageSource) Collect(_ context.Context, t usagesync.Target, _ usagesync.Window) (usagesync.Result, error) {
	f.mu.Lock()
	f.targets = append(f.targets, t)
	f.mu.Unlock()
	if err := f.fail[t.ID]; err != nil {
		return usagesync.Result{}, err
	}
	return usagesync.Result{
		Values: map[string]float64{"standard_storage_gb": f.gb[t.ID]},
		Series: map[string]map[string]float64{"standard_storage_gb": {"2026-09": f.gb[t.ID]}},
	}, nil
}

func useFakeSource(t *testing.T, f *fakeUsageSource) {
	t.Helper()
	old := newUsageSource
	newUsageSource = func(context.Context, string) (usagesync.Source, error) { return f, nil }
	t.Cleanup(func() { newUsageSource = old })
}

func fileHash(t *testing.T, path string) [32]byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(raw)
}

// The point of the feature: an estimate priced before and after a sync
// differs on the usage-driven line, with the snapshot found by default.
func TestUsageSyncChangesTheEstimate(t *testing.T) {
	dir := writeBucketProject(t)
	useFakeSource(t, &fakeUsageSource{gb: map[string]float64{"acme-data": 500}})

	before := bucketSubtotal(t)
	out, err := runCLI(t, "usage", "sync")
	if err != nil {
		t.Fatalf("usage sync: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Synced 1 resource(s) from aws over 30 days → c3x-usage.synced.yml") {
		t.Errorf("summary missing:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "c3x-usage.synced.yml")); err != nil {
		t.Fatalf("snapshot not written: %v", err)
	}

	if after := bucketSubtotal(t); after == before {
		t.Errorf("subtotal stayed %s after syncing 500 GB of storage", after)
	}
}

// A re-sync can never clobber a hand edit: sync does not read or write the
// hand-written file.
func TestUsageSyncLeavesTheHandWrittenFileByteIdentical(t *testing.T) {
	dir := writeBucketProject(t)
	writeProjectFile(t, dir, ".c3x.toml", `usage_path = "c3x-usage.yml"`+"\n")
	hand := filepath.Join(dir, "c3x-usage.yml")
	writeProjectFile(t, dir, "c3x-usage.yml", "# mine\nversion: \"0.1\"\nresource_usage:\n  aws_s3_bucket.data:\n    standard_storage_gb: 10\n")
	want := fileHash(t, hand)

	useFakeSource(t, &fakeUsageSource{gb: map[string]float64{"acme-data": 500}})
	for i := 0; i < 2; i++ {
		if out, err := runCLI(t, "usage", "sync"); err != nil {
			t.Fatalf("run %d: %v\n%s", i, err, out)
		}
		if got := fileHash(t, hand); got != want {
			t.Fatalf("run %d changed the hand-written usage file", i)
		}
	}
}

func TestUsageSyncRefusesToOverwriteTheHandWrittenFile(t *testing.T) {
	dir := writeBucketProject(t)
	writeProjectFile(t, dir, ".c3x.toml", `usage_path = "c3x-usage.yml"`+"\n")
	writeProjectFile(t, dir, "c3x-usage.yml", "version: \"0.1\"\n")
	want := fileHash(t, filepath.Join(dir, "c3x-usage.yml"))
	fake := &fakeUsageSource{}
	useFakeSource(t, fake)

	out, err := runCLI(t, "usage", "sync", "--out", "c3x-usage.yml")
	if err == nil || !strings.Contains(err.Error(), "hand-written usage file") {
		t.Fatalf("err = %v, want a refusal\n%s", err, out)
	}
	if fileHash(t, filepath.Join(dir, "c3x-usage.yml")) != want || len(fake.targets) != 0 {
		t.Error("the hand-written file was touched or the cloud was called")
	}
}

// Untrusted-input mode must never reach the cloud with the runner's
// credentials, however it is switched on.
func TestUsageSyncRefusesInUntrustedMode(t *testing.T) {
	writeBucketProject(t)
	fake := &fakeUsageSource{gb: map[string]float64{"acme-data": 1}}
	useFakeSource(t, fake)

	t.Setenv("C3X_NO_REMOTE_MODULES", "1")
	out, err := runCLI(t, "usage", "sync")
	if err == nil || !strings.Contains(err.Error(), "untrusted-input mode") {
		t.Fatalf("err = %v, want a refusal\n%s", err, out)
	}
	if len(fake.targets) != 0 {
		t.Error("the source was called")
	}
	if _, statErr := os.Stat("c3x-usage.synced.yml"); statErr == nil {
		t.Error("a snapshot was written")
	}
}

func TestUsageSyncRefusesWhenTheProjectConfigTurnsUntrustedModeOn(t *testing.T) {
	dir := writeBucketProject(t)
	writeProjectFile(t, dir, ".c3x.toml", "no_remote_modules = true\n")
	fake := &fakeUsageSource{}
	useFakeSource(t, fake)

	if _, err := runCLI(t, "usage", "sync"); err == nil || !strings.Contains(err.Error(), "untrusted-input mode") {
		t.Fatalf("err = %v, want a refusal", err)
	}
	if len(fake.targets) != 0 {
		t.Error("the source was called")
	}
}

// The state is the source of truth for the real name.
func TestUsageSyncResolvesTheBucketNameFromState(t *testing.T) {
	dir := writeBucketProject(t)
	writeProjectFile(t, dir, "state.json", `{"values":{"root_module":{"resources":[{
		"address":"aws_s3_bucket.data","mode":"managed","type":"aws_s3_bucket","name":"data",
		"values":{"bucket":"acme-data-7f3a","region":"ap-northeast-1"}}]}}}`)
	fake := &fakeUsageSource{gb: map[string]float64{"acme-data-7f3a": 42}}
	useFakeSource(t, fake)

	if out, err := runCLI(t, "usage", "sync", "--state", "state.json"); err != nil {
		t.Fatalf("usage sync: %v\n%s", err, out)
	}
	if len(fake.targets) != 1 || fake.targets[0].ID != "acme-data-7f3a" || fake.targets[0].Region != "ap-northeast-1" {
		t.Errorf("targets = %+v, want the state's name and region", fake.targets)
	}
}

// One failure is recorded and skipped; --strict turns it into an exit
// code but the file is still written.
func TestUsageSyncPartialFailure(t *testing.T) {
	dir := writeBucketProject(t)
	writeProjectFile(t, dir, "main.tf", `
		provider "aws" { region = "us-east-1" }
		resource "aws_s3_bucket" "data" { bucket = "acme-data" }
		resource "aws_s3_bucket" "logs" { bucket = "acme-logs" }
	`)
	useFakeSource(t, &fakeUsageSource{
		gb:   map[string]float64{"acme-data": 7},
		fail: map[string]error{"acme-logs": errors.New("AccessDenied")},
	})

	out, err := runCLI(t, "usage", "sync")
	if err != nil {
		t.Fatalf("usage sync: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Synced 1 resource(s)") || !strings.Contains(out, "skipped aws_s3_bucket.logs: AccessDenied") {
		t.Errorf("output:\n%s", out)
	}

	out, err = runCLI(t, "usage", "sync", "--strict")
	if err == nil || !strings.Contains(err.Error(), "1 resource(s) could not be synced") {
		t.Fatalf("--strict err = %v\n%s", err, out)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "c3x-usage.synced.yml"))
	if !strings.Contains(string(raw), "aws_s3_bucket.data:") {
		t.Errorf("the snapshot was not written under --strict:\n%s", raw)
	}
	if !strings.Contains(string(raw), "AccessDenied") {
		t.Errorf("the error was not recorded in the snapshot:\n%s", raw)
	}
}

func TestUsageSyncSaysWhenThereIsNothingToSync(t *testing.T) {
	dir := writeBucketProject(t)
	writeProjectFile(t, dir, "main.tf", `resource "aws_vpc" "v" { cidr_block = "10.0.0.0/16" }`+"\n")
	useFakeSource(t, &fakeUsageSource{})

	out, err := runCLI(t, "usage", "sync")
	if err != nil {
		t.Fatalf("usage sync: %v\n%s", err, out)
	}
	if !strings.Contains(out, "No resources of a supported kind (aws_s3_bucket)") {
		t.Errorf("output:\n%s", out)
	}
}

func TestUsageSyncRejectsAnUnsupportedProviderAndBadDays(t *testing.T) {
	writeBucketProject(t)
	if _, err := runCLI(t, "usage", "sync", "--provider", "azure"); err == nil || !strings.Contains(err.Error(), "unsupported provider") {
		t.Errorf("provider err = %v", err)
	}
	if _, err := runCLI(t, "usage", "sync", "--days", "0"); err == nil || !strings.Contains(err.Error(), "--days") {
		t.Errorf("days err = %v", err)
	}
}

// A run that measured nothing must not replace a good snapshot with an
// empty one: no credentials, a forgotten --state or a missing permission
// would otherwise silently erase the data.
func TestUsageSyncKeepsTheExistingSnapshotWhenNothingWasMeasured(t *testing.T) {
	dir := writeBucketProject(t)
	synced := filepath.Join(dir, "c3x-usage.synced.yml")
	writeProjectFile(t, dir, "c3x-usage.synced.yml", syncedBucket)
	want := fileHash(t, synced)

	// Every lookup fails.
	useFakeSource(t, &fakeUsageSource{fail: map[string]error{"acme-data": errors.New("no credentials")}})
	out, err := runCLI(t, "usage", "sync")
	if err == nil || !strings.Contains(err.Error(), "no resource could be synced") {
		t.Fatalf("err = %v, want a failure\n%s", err, out)
	}
	if !strings.Contains(out, "skipped aws_s3_bucket.data: no credentials") {
		t.Errorf("the reason was not shown:\n%s", out)
	}
	if fileHash(t, synced) != want {
		t.Error("the existing snapshot was replaced")
	}

	// Nothing could even be looked up: the name is not a literal and there
	// is no --state.
	writeProjectFile(t, dir, "main.tf", `
		provider "aws" { region = "us-east-1" }
		resource "aws_s3_bucket" "data" { bucket_prefix = "logs-" }
	`)
	out, err = runCLI(t, "usage", "sync")
	if err == nil || !strings.Contains(out, "pass --state") {
		t.Fatalf("err = %v, want a failure naming --state\n%s", err, out)
	}
	if fileHash(t, synced) != want {
		t.Error("the existing snapshot was replaced")
	}
}

// A project with no resource of a supported kind is a legitimate empty
// result, and the snapshot reflects it.
func TestUsageSyncWritesAnEmptySnapshotWhenThereIsNothingToLookUp(t *testing.T) {
	dir := writeBucketProject(t)
	writeProjectFile(t, dir, "main.tf", `resource "aws_vpc" "v" { cidr_block = "10.0.0.0/16" }`+"\n")
	writeProjectFile(t, dir, "c3x-usage.synced.yml", syncedBucket)
	useFakeSource(t, &fakeUsageSource{})

	if out, err := runCLI(t, "usage", "sync"); err != nil {
		t.Fatalf("usage sync: %v\n%s", err, out)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "c3x-usage.synced.yml"))
	if strings.Contains(string(raw), "acme-data") || !strings.Contains(string(raw), "resource_usage: {}") {
		t.Errorf("the snapshot should now be empty:\n%s", raw)
	}
}
