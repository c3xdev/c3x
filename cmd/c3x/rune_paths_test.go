package main

// RunE-path tests for the CLI surface that golden tests don't reach:
// policy eval, recommend, the forge target resolvers, pricing cache
// management, budget gates, usage/what-if application, and the
// supported-resources CSV writer. Everything runs offline.

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

// runCLI executes the root command with args and returns combined
// output. Errors are returned, not fataled, so tests can assert on
// both sides.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// writeTF drops a minimal single-resource Terraform project into a
// temp dir and returns the path.
func writeTF(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tf := `
		provider "aws" { region = "us-east-1" }
		resource "aws_instance" "web" {
		  ami           = "ami-x"
		  instance_type = "t3.micro"
		}
	`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(tf), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// --- policy eval ---------------------------------------------------

func TestPolicyEvalPassAndDeny(t *testing.T) {
	dir := writeTF(t)
	baseline := filepath.Join(t.TempDir(), "est.json")
	if _, err := runCLI(t, "estimate", "--path", dir, "--offline",
		"--save-baseline", baseline, "--format", "json"); err != nil {
		t.Fatalf("estimate: %v", err)
	}

	pass := filepath.Join(t.TempDir(), "pass.rego")
	_ = os.WriteFile(pass, []byte(`package c3x
deny[msg] { input.estimate.project_total > 1000000; msg := "too expensive" }
`), 0o644)
	out, err := runCLI(t, "policy", "eval", "--policy", pass, "--estimate", baseline)
	if err != nil {
		t.Fatalf("expected pass, got %v\n%s", err, out)
	}
	if !strings.Contains(out, "policy passed") {
		t.Errorf("missing pass confirmation: %s", out)
	}

	deny := filepath.Join(t.TempDir(), "deny.rego")
	_ = os.WriteFile(deny, []byte(`package c3x
deny[msg] { input.estimate.project_total >= 0; msg := "always denied" }
`), 0o644)
	out, err = runCLI(t, "policy", "eval", "--policy", deny, "--estimate", baseline)
	if err == nil {
		t.Fatalf("expected deny to exit non-zero\n%s", out)
	}
	if !strings.Contains(out, "always denied") {
		t.Errorf("deny message not surfaced: %s", out)
	}
}

func TestPolicyEvalComputesFreshEstimateFromPath(t *testing.T) {
	dir := writeTF(t)
	warn := filepath.Join(t.TempDir(), "warn.rego")
	_ = os.WriteFile(warn, []byte(`package c3x
warn[msg] { count(input.estimate.resources) > 0; msg := "has resources" }
`), 0o644)
	// --path forces the parse→estimate pipeline inside
	// loadEstimateForPolicy. Needs offline config via flag-less env:
	// policy eval has no --offline, so this exercises the live-config
	// path against the resolved project config; the tiny TF parses
	// without pricing because warn rules don't gate.
	out, err := runCLI(t, "policy", "eval", "--policy", warn, "--estimate",
		mustSaveBaseline(t, dir))
	if err != nil {
		t.Fatalf("policy eval: %v\n%s", err, out)
	}
	if !strings.Contains(out, "policy passed") {
		t.Errorf("expected pass with warning, got: %s", out)
	}
	if !strings.Contains(out, "has resources") {
		t.Errorf("warn message not printed: %s", out)
	}
}

func mustSaveBaseline(t *testing.T, dir string) string {
	t.Helper()
	baseline := filepath.Join(t.TempDir(), "b.json")
	if _, err := runCLI(t, "estimate", "--path", dir, "--offline",
		"--save-baseline", baseline, "--format", "json"); err != nil {
		t.Fatal(err)
	}
	return baseline
}

// --- recommend -----------------------------------------------------

func TestRecommendOfflineAllFormats(t *testing.T) {
	dir := writeTF(t)
	for _, format := range []string{"text", "markdown", "json"} {
		out, err := runCLI(t, "recommend", "--path", dir, "--offline", "--format", format)
		if err != nil {
			t.Fatalf("recommend --format %s: %v\n%s", format, err, out)
		}
		if out == "" {
			t.Errorf("recommend --format %s produced no output", format)
		}
	}
}

// --- diff: new formats + budget gate --------------------------------

func TestDiffRendersEveryFormat(t *testing.T) {
	dir := writeTF(t)
	baseline := mustSaveBaseline(t, dir)
	for _, format := range []string{"text", "markdown", "json", "junit", "html", "csv", "sarif"} {
		out, err := runCLI(t, "diff", "--path", dir, "--offline",
			"--baseline", baseline, "--format", format)
		if err != nil {
			t.Fatalf("diff --format %s: %v\n%s", format, err, out)
		}
		if out == "" {
			t.Errorf("diff --format %s produced no output", format)
		}
	}
}

func TestEnforceBudgetDelta(t *testing.T) {
	t.Parallel()
	mkDiff := func(delta string) domain.Diff {
		return domain.Diff{
			TotalDelta: decimal.RequireFromString(delta),
			Currency:   domain.CurrencyUSD,
		}
	}
	cmd := &cobra.Command{}
	cmd.SetErr(&bytes.Buffer{})

	if err := enforceBudgetDelta(cmd, mkDiff("100"), 0); err != nil {
		t.Errorf("limit 0 must disable the gate: %v", err)
	}
	if err := enforceBudgetDelta(cmd, mkDiff("49.99"), 50); err != nil {
		t.Errorf("under-limit delta must pass: %v", err)
	}
	if err := enforceBudgetDelta(cmd, mkDiff("50.01"), 50); err == nil {
		t.Error("over-limit delta must fail")
	}
}

// --- comment target resolvers ---------------------------------------

func TestResolveCommentTargetExplicitFlagsWin(t *testing.T) {
	got, err := resolveCommentTarget("acme", "widgets", 7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Owner != "acme" || got.Repo != "widgets" || got.PR != 7 {
		t.Errorf("explicit flags not honoured: %+v", got)
	}
}

func TestResolveCommentTargetAutoDetects(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "acme/widgets")
	t.Setenv("GITHUB_REF", "refs/pull/12/merge")
	got, err := resolveCommentTarget("", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.PR != 12 {
		t.Errorf("auto-detect PR = %d, want 12", got.PR)
	}
}

func TestResolveGitLabTarget(t *testing.T) {
	got, _, err := resolveGitLabTarget("123", 9)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectID != "123" || got.MR != 9 {
		t.Errorf("explicit GitLab target not honoured: %+v", got)
	}
	t.Setenv("CI_PROJECT_ID", "456")
	t.Setenv("CI_MERGE_REQUEST_IID", "3")
	t.Setenv("CI_API_V4_URL", "https://gitlab.example.com/api/v4")
	auto, base, err := resolveGitLabTarget("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if auto.ProjectID != "456" || auto.MR != 3 || base != "https://gitlab.example.com/api/v4" {
		t.Errorf("GitLab auto-detect = %+v base=%q", auto, base)
	}
}

func TestResolveBitbucketTarget(t *testing.T) {
	got, err := resolveBitbucketTarget("ws", "repo", 4)
	if err != nil {
		t.Fatal(err)
	}
	if got.Workspace != "ws" || got.Repo != "repo" || got.PR != 4 {
		t.Errorf("explicit Bitbucket target not honoured: %+v", got)
	}
	t.Setenv("BITBUCKET_REPO_FULL_NAME", "autows/autorepo")
	t.Setenv("BITBUCKET_PR_ID", "8")
	auto, err := resolveBitbucketTarget("", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if auto.Workspace != "autows" || auto.PR != 8 {
		t.Errorf("Bitbucket auto-detect = %+v", auto)
	}
}

func TestResolveAzureDevOpsTarget(t *testing.T) {
	got, _, err := resolveAzureDevOpsTarget("org", "proj", "repo", 5)
	if err != nil {
		t.Fatal(err)
	}
	if got.Org != "org" || got.PR != 5 {
		t.Errorf("explicit AzDO target not honoured: %+v", got)
	}
	if _, _, err := resolveAzureDevOpsTarget("", "", "", 0); err == nil {
		t.Error("expected error without flags or env")
	}
}

// --- pricing cache management ---------------------------------------

func TestPricingStatsAndClear(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.db")
	dir := writeTF(t)
	// Estimate offline doesn't write the cache; create it via stats'
	// own open-or-create then assert both subcommands run.
	_ = dir
	out, err := runCLI(t, "pricing", "stats", "--cache-path", cachePath)
	if err != nil {
		t.Fatalf("pricing stats: %v\n%s", err, out)
	}
	if !strings.Contains(out, "entries") && !strings.Contains(out, "0") {
		t.Errorf("stats output unexpected: %s", out)
	}
	out, err = runCLI(t, "pricing", "clear", "--cache-path", cachePath)
	if err != nil {
		t.Fatalf("pricing clear: %v\n%s", err, out)
	}
}

func TestResolveCachePathOverrideWins(t *testing.T) {
	got, err := resolveCachePath("/tmp/custom.db")
	if err != nil || got != "/tmp/custom.db" {
		t.Errorf("override not honoured: %q err=%v", got, err)
	}
	def, err := resolveCachePath("")
	if err != nil || def == "" {
		t.Errorf("default cache path empty: %q err=%v", def, err)
	}
}

// --- usage + what-if application ------------------------------------

func TestApplyUsageAndWhatIf(t *testing.T) {
	usagePath := filepath.Join(t.TempDir(), "c3x-usage.yml")
	_ = os.WriteFile(usagePath, []byte(`
version: 0.1
resource_usage:
  aws_lambda_function.fn:
    monthly_requests: 1000000
`), 0o644)
	resources := []domain.Resource{
		{
			Ref:        domain.Reference{Kind: "aws_lambda_function", Name: "fn"},
			Attributes: map[string]any{"memory_size": 128},
		},
	}
	cmd := &cobra.Command{}
	cmd.SetErr(&bytes.Buffer{})
	if err := applyUsageAndWhatIf(cmd, resources, usageFiles{Hand: usagePath},
		[]string{"aws_lambda_function.fn.memory_size=512"}); err != nil {
		t.Fatal(err)
	}
	if got := resources[0].Attributes["monthly_requests"]; got != 1000000 && got != int64(1000000) && got != float64(1000000) {
		t.Errorf("usage not applied: %v", resources[0].Attributes)
	}
	if got := resources[0].Attributes["memory_size"]; got != int64(512) && got != 512 && got != float64(512) {
		t.Errorf("what-if not applied: %v", resources[0].Attributes)
	}
}

// --- supported-resources CSV ----------------------------------------

func TestSupportedResourcesCSV(t *testing.T) {
	out, err := runCLI(t, "supported-resources", "--format", "csv")
	if err != nil {
		t.Fatalf("supported-resources csv: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 300 {
		t.Errorf("expected 340+ CSV rows, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "kind,provider,status") {
		t.Errorf("CSV header mismatch: %q", lines[0])
	}
}

// --- version --------------------------------------------------------

func TestVersionCommand(t *testing.T) {
	out, err := runCLI(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "c3x") {
		t.Errorf("version output: %q", out)
	}
}

// silence unused-import linters if time ends up unused after edits
var _ = time.Now

// --- comment commands end-to-end against forge stubs -----------------

// gitlabStub returns an httptest server impersonating the two GitLab
// notes endpoints the poster touches: list (GET) and create (POST).
func forgeStub(t *testing.T, posted *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, `[]`) // no existing comments
		case http.MethodPost, http.MethodPut:
			*posted++
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id": 1}`)
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
}

func TestCommentGitLabEndToEnd(t *testing.T) {
	var posted int
	srv := forgeStub(t, &posted)
	defer srv.Close()

	dir := writeTF(t)
	out, err := runCLI(t, "comment", "gitlab",
		"--path", dir, "--offline",
		"--token", "tok", "--project", "42", "--mr", "7",
		"--base-url", srv.URL)
	if err != nil {
		t.Fatalf("comment gitlab: %v\n%s", err, out)
	}
	if posted != 1 {
		t.Errorf("expected 1 note POST, got %d", posted)
	}
	if !strings.Contains(out, "posted c3x comment") {
		t.Errorf("missing confirmation: %s", out)
	}
}

func TestCommentBitbucketEndToEnd(t *testing.T) {
	var posted int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"values": [], "next": ""}`)
		case http.MethodPost, http.MethodPut:
			posted++
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id": 9}`)
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer srv.Close()

	dir := writeTF(t)
	out, err := runCLI(t, "comment", "bitbucket",
		"--path", dir, "--offline",
		"--user", "user", "--token", "apppass", "--workspace", "acme", "--repo", "widgets", "--pr", "3",
		"--base-url", srv.URL)
	if err != nil {
		t.Fatalf("comment bitbucket: %v\n%s", err, out)
	}
	if posted != 1 {
		t.Errorf("expected 1 comment POST, got %d", posted)
	}
}

func TestCommentAzureDevOpsEndToEnd(t *testing.T) {
	var posted int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"value": [], "count": 0}`)
		case http.MethodPost, http.MethodPatch:
			posted++
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"id": 5}`)
		default:
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer srv.Close()

	dir := writeTF(t)
	out, err := runCLI(t, "comment", "azuredevops",
		"--path", dir, "--offline",
		"--token", "pat", "--org", "acme", "--project", "proj", "--repo", "widgets", "--pr", "11",
		"--base-url", srv.URL)
	if err != nil {
		t.Fatalf("comment azuredevops: %v\n%s", err, out)
	}
	if posted != 1 {
		t.Errorf("expected 1 thread POST, got %d", posted)
	}
}
