package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// flatPricing answers every price lookup with $1/hr, so any priced
// resource has a non-zero cost without the network.
func flatPricing(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if !strings.HasSuffix(r.URL.Path, "/graphql") {
			http.NotFound(w, r) // catalog fetch: fall back to the embedded one
			return
		}
		_, _ = io.WriteString(w, `{"data":{"products":[{"prices":[{"USD":"1.00","unit":"Hrs"}]}]}}`)
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/graphql"
}

func writePlan(t *testing.T, before, after string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "plan.json")
	raw := `{"resource_changes":[{"address":"aws_instance.web","type":"aws_instance","name":"web",
		"change":{"actions":["` + map[bool]string{true: "create", false: "update"}[before == "null"] + `"],
		"before":` + before + `,"after":` + after + `}}]}`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// With a plan JSON, `c3x diff` needs no --baseline: the plan's prior
// state is the baseline. The Action relies on this for budget-delta,
// because a plan file is a build artifact the base branch doesn't have.
func TestDiffPlanWithoutBaselineGatesTheIncrease(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	endpoint := flatPricing(t)
	created := writePlan(t, "null", `{"instance_type":"m5.xlarge"}`)

	out, err := runCLI(t, "diff", "--path", created, "--pricing-endpoint", endpoint,
		"--no-cache", "--budget-delta", "1")
	if !errors.Is(err, errBudgetDeltaExceeded) {
		t.Fatalf("a plan creating an instance must trip --budget-delta 1; err = %v\n%s", err, out)
	}
	if !strings.Contains(out, "aws_instance.web") {
		t.Errorf("diff should list the created instance:\n%s", out)
	}
}

func TestDiffPlanWithoutBaselineUnchangedCostPasses(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	endpoint := flatPricing(t)
	// Same flat price before and after: a zero delta.
	updated := writePlan(t, `{"instance_type":"m5.large"}`, `{"instance_type":"m5.xlarge"}`)

	if out, err := runCLI(t, "diff", "--path", updated, "--pricing-endpoint", endpoint,
		"--no-cache", "--budget-delta", "1"); err != nil {
		t.Fatalf("a zero-delta plan must pass the gate: %v\n%s", err, out)
	}
}

func TestDiffDirectoryStillNeedsBaseline(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := runCLI(t, "diff", "--path", t.TempDir(), "--offline")
	if err == nil || !strings.Contains(err.Error(), "--baseline is required") {
		t.Fatalf("err = %v, want the --baseline requirement for a directory", err)
	}
}
