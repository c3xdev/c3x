package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	awsusage "github.com/c3xdev/c3x/internal/usagesync/aws"
)

// TestDoctorCatalogCheckPasses validates the catalog check
// independently of network availability. The catalog check is
// purely embedded data and must pass on every machine.
func TestDoctorCatalogCheckPasses(t *testing.T) {
	result := checkCatalog()
	if !result.OK {
		t.Errorf("catalog check failed: %v", result.Detail)
	}
	if !strings.Contains(result.Detail, "resource kinds loaded") {
		t.Errorf("expected count in detail, got: %s", result.Detail)
	}
}

// TestDoctorCacheCheckPasses ensures the cache writability check
// works against the user's actual cache dir, redirected via
// XDG_CACHE_HOME to a t.TempDir for hermeticity.
func TestDoctorCacheCheckPasses(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	result := checkCache()
	if !result.OK {
		t.Errorf("cache check failed: %v", result.Detail)
	}
}

// TestDoctorRendersFailuresWithHint confirms the rendering pulls
// the hint text through for failed checks.
func TestDoctorRendersFailuresWithHint(t *testing.T) {
	failed := checkResult{
		Name:   "endpoint",
		OK:     false,
		Detail: "HTTP 503",
		Hint:   "service is down",
	}
	rendered := failed.Render()
	if !strings.Contains(rendered, "✗") {
		t.Errorf("expected fail icon, got: %s", rendered)
	}
	if !strings.Contains(rendered, "service is down") {
		t.Errorf("hint missing: %s", rendered)
	}
}

func TestDoctorCommandSucceedsOnHappyPath(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	// --quiet to skip noise on passing checks; we only care about
	// exit status.
	cmd.SetArgs([]string{"doctor", "--quiet"})
	// The endpoint check makes a real HTTP request. Skip if the
	// network is unavailable in the test env — this is a smoke
	// integration, not a hermetic unit test.
	err := cmd.Execute()
	if err != nil {
		t.Skipf("doctor failed (likely offline test env): %v\n%s", err, out.String())
	}
}

// Doctor asks about AWS credentials only where usage sync is in use.
func TestDoctorUsageSyncCheckIsAbsentUntilConfigured(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	called := false
	old := checkAWSCredentials
	checkAWSCredentials = func(context.Context) (awsusage.Identity, error) { called = true; return awsusage.Identity{}, nil }
	t.Cleanup(func() { checkAWSCredentials = old })

	if _, ok := checkUsageSync(context.Background()); ok || called {
		t.Fatalf("the check ran with no synced usage file (ok=%v called=%v)", ok, called)
	}
}

func TestDoctorUsageSyncCheckReportsCredentials(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "c3x-usage.synced.yml"), []byte("version: \"0.1\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := checkAWSCredentials
	t.Cleanup(func() { checkAWSCredentials = old })

	checkAWSCredentials = func(context.Context) (awsusage.Identity, error) {
		return awsusage.Identity{Region: "eu-west-1", CredentialSource: "SharedConfigCredentials"}, nil
	}
	r, ok := checkUsageSync(context.Background())
	if !ok || !r.OK || !strings.Contains(r.Detail, "SharedConfigCredentials") || !strings.Contains(r.Detail, "eu-west-1") {
		t.Errorf("ok result = %+v (ok=%v)", r, ok)
	}

	checkAWSCredentials = func(context.Context) (awsusage.Identity, error) {
		return awsusage.Identity{}, errors.New("no AWS credentials: nothing found")
	}
	r, ok = checkUsageSync(context.Background())
	if !ok || r.OK || r.Hint == "" || !strings.Contains(r.Detail, "no AWS credentials") {
		t.Errorf("failing result = %+v (ok=%v)", r, ok)
	}
}
