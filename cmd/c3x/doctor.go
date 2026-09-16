package main

// `c3x doctor` runs pre-flight checks and reports the health of
// every subsystem the CLI depends on. Each check is independent —
// one failure doesn't abort the others — so users see the full
// picture in a single invocation.
//
// Exits non-zero if any check fails so it's usable as a CI gate.
//
// Checks:
//   1. catalog loads (embedded TOMLs parse + validate)
//   2. pricing endpoint reachable (HTTP 200 to a probe query)
//   3. cache directory writable (creates + deletes a probe file)
//   4. config resolves (verifies the user's resolved-config is parseable)

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	var quiet bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run pre-flight checks (catalog, endpoint, cache, config).",
		Long: `Each subsystem the CLI depends on gets a status check. Exits
non-zero if any check fails so the command is usable as a CI gate.

  catalog    — embedded TOMLs parse and validate
  endpoint   — pricing.c3x.dev responds to a probe query
  cache      — local cache directory is writable
  config     — user config resolves without errors`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			results := runDoctorChecks(cmd.Context())
			anyFailed := false
			for _, r := range results {
				if !quiet || !r.OK {
					fmt.Fprintln(cmd.OutOrStdout(), r.Render())
				}
				if !r.OK {
					anyFailed = true
				}
			}
			if anyFailed {
				return errors.New("one or more doctor checks failed")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "\nAll checks passed.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&quiet, "quiet", false, "print only failed checks")
	return cmd
}

// checkResult is one row in the doctor's output. We split passing
// from failing rendering so users can scan a long output quickly.
type checkResult struct {
	Name   string
	OK     bool
	Detail string
	Hint   string // optional one-liner with the next step a user can take
}

func (r checkResult) Render() string {
	icon := "✓"
	if !r.OK {
		icon = "✗"
	}
	line := fmt.Sprintf("%s  %-12s  %s", icon, r.Name, r.Detail)
	if !r.OK && r.Hint != "" {
		line += "\n   ↳ " + r.Hint
	}
	return line
}

func runDoctorChecks(ctx context.Context) []checkResult {
	return []checkResult{
		checkCatalog(),
		checkEndpoint(ctx),
		checkCache(),
		checkConfig(),
	}
}

func checkCatalog() checkResult {
	reg, err := catalog.Load()
	if err != nil {
		return checkResult{
			Name:   "catalog",
			OK:     false,
			Detail: fmt.Sprintf("Load failed: %v", err),
			Hint:   "this is a c3x build defect — embedded TOMLs should always parse",
		}
	}
	kinds := reg.Kinds()
	return checkResult{
		Name:   "catalog",
		OK:     true,
		Detail: fmt.Sprintf("%d resource kinds loaded from embedded TOMLs", len(kinds)),
	}
}

// checkEndpoint POSTs a tiny GraphQL probe to the configured pricing
// endpoint. We don't care about the response shape, only that the
// HTTP layer round-trips. A 200 with empty result is good; anything
// else (timeout, 5xx, DNS) fails the check.
func checkEndpoint(ctx context.Context) checkResult {
	resolved, err := config.Resolve("", nil)
	if err != nil {
		return checkResult{
			Name:   "endpoint",
			OK:     false,
			Detail: fmt.Sprintf("config.Resolve failed: %v", err),
			Hint:   "fix the config issue surfaced by `c3x doctor`'s config check",
		}
	}
	endpoint := resolved.PricingEndpoint
	if endpoint == "" {
		endpoint = pricing.DefaultEndpoint
	}
	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	// Minimal GraphQL POST — the upstream returns errors on an empty
	// query but the HTTP layer still 200s, which is all we want.
	body := strings.NewReader(`{"query":"{ __typename }"}`)
	// DefaultEndpoint already includes the /graphql path. Tolerate
	// either form so users can set --pricing-endpoint to a bare host
	// or a path-included URL.
	probeURL := endpoint
	if !strings.HasSuffix(probeURL, "/graphql") {
		probeURL = strings.TrimRight(probeURL, "/") + "/graphql"
	}
	req, err := http.NewRequestWithContext(probeCtx, http.MethodPost, probeURL, body)
	if err != nil {
		return checkResult{
			Name:   "endpoint",
			OK:     false,
			Detail: fmt.Sprintf("constructing request: %v", err),
		}
	}
	req.Header.Set("Content-Type", "application/json")
	if resolved.PricingToken != "" {
		req.Header.Set("Authorization", "Bearer "+resolved.PricingToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return checkResult{
			Name:   "endpoint",
			OK:     false,
			Detail: fmt.Sprintf("HTTP error: %v", err),
			Hint:   "is the network up? is " + endpoint + " reachable?",
		}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return checkResult{
			Name:   "endpoint",
			OK:     false,
			Detail: fmt.Sprintf("HTTP %d from %s", resp.StatusCode, endpoint),
			Hint:   "endpoint reachable but returned a non-200; check service status",
		}
	}
	return checkResult{
		Name:   "endpoint",
		OK:     true,
		Detail: fmt.Sprintf("%s — HTTP 200", endpoint),
	}
}

// checkCache writes a probe file to the resolved cache directory
// and removes it. Surfaces permission issues users would otherwise
// only hit at first run.
func checkCache() checkResult {
	dir, err := config.UserCachePath()
	if err != nil {
		return checkResult{
			Name:   "cache",
			OK:     false,
			Detail: fmt.Sprintf("UserCachePath: %v", err),
			Hint:   "set $XDG_CACHE_HOME or $HOME to a writable path",
		}
	}
	cacheDir := filepath.Dir(dir)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return checkResult{
			Name:   "cache",
			OK:     false,
			Detail: fmt.Sprintf("mkdir %s: %v", cacheDir, err),
		}
	}
	probe := filepath.Join(cacheDir, ".c3x-doctor-probe")
	if err := os.WriteFile(probe, []byte("probe"), 0o600); err != nil {
		return checkResult{
			Name:   "cache",
			OK:     false,
			Detail: fmt.Sprintf("write %s: %v", probe, err),
			Hint:   "check filesystem permissions on " + cacheDir,
		}
	}
	_ = os.Remove(probe)
	return checkResult{
		Name:   "cache",
		OK:     true,
		Detail: fmt.Sprintf("%s writable", cacheDir),
	}
}

func checkConfig() checkResult {
	if _, err := config.Resolve("", nil); err != nil {
		return checkResult{
			Name:   "config",
			OK:     false,
			Detail: fmt.Sprintf("Resolve failed: %v", err),
		}
	}
	return checkResult{
		Name:   "config",
		OK:     true,
		Detail: "user config resolved (defaults applied where unset)",
	}
}
