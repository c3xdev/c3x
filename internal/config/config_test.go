package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/domain"
)

func TestResolveDefaultsWhenNoFilesPresent(t *testing.T) {
	// t.Setenv is incompatible with t.Parallel; tests that need to
	// mutate env vars run sequentially.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()

	got, err := config.Resolve(dir, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	defaults := config.Defaults()
	if got.Region != defaults.Region {
		t.Errorf("Region = %q, want default %q", got.Region, defaults.Region)
	}
	if got.PricingEndpoint != defaults.PricingEndpoint {
		t.Errorf("PricingEndpoint = %q, want default %q",
			got.PricingEndpoint, defaults.PricingEndpoint)
	}
	if got.Currency != domain.CurrencyUSD {
		t.Errorf("Currency = %v, want USD", got.Currency)
	}
}

func TestResolveProjectFileOverridesDefaults(t *testing.T) {
	// t.Setenv is incompatible with t.Parallel; tests that mutate env
	// vars run sequentially.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, ".c3x.toml"), []byte(`
region = "eu-west-1"
format = "markdown"
`), 0o644))

	got, err := config.Resolve(dir, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Region != "eu-west-1" {
		t.Errorf("Region = %q, want eu-west-1", got.Region)
	}
	if got.Format != "markdown" {
		t.Errorf("Format = %q, want markdown", got.Format)
	}
}

func TestResolveEnvOverridesProjectFile(t *testing.T) {
	// t.Setenv is incompatible with t.Parallel; tests that mutate env
	// vars run sequentially.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, ".c3x.toml"), []byte(`
region = "eu-west-1"
`), 0o644))
	t.Setenv("C3X_REGION", "us-west-2")

	got, err := config.Resolve(dir, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Region != "us-west-2" {
		t.Errorf("Region = %q, want us-west-2 (env should beat file)", got.Region)
	}
}

func TestResolvePricingTokenDefaultsEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := config.Resolve(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.PricingToken != "" {
		t.Errorf("PricingToken = %q, want empty by default (public endpoint needs no auth)", got.PricingToken)
	}
}

func TestResolvePricingTokenFromEnv(t *testing.T) {
	// t.Setenv is incompatible with t.Parallel; runs sequentially.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("C3X_PRICING_TOKEN", "self-hosted-key")

	got, err := config.Resolve(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.PricingToken != "self-hosted-key" {
		t.Errorf("PricingToken = %q, want %q from C3X_PRICING_TOKEN", got.PricingToken, "self-hosted-key")
	}
}

func TestResolveFlagsOverrideEverything(t *testing.T) {
	// t.Setenv is incompatible with t.Parallel; tests that mutate env
	// vars run sequentially.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, ".c3x.toml"), []byte(`
region = "eu-west-1"
`), 0o644))
	t.Setenv("C3X_REGION", "us-west-2")

	got, err := config.Resolve(dir, map[string]any{"region": "ap-south-1"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Region != "ap-south-1" {
		t.Errorf("Region = %q, want ap-south-1 (flag should beat env)", got.Region)
	}
}

func TestValidateRejectsBadFormat(t *testing.T) {
	t.Parallel()

	r := config.Defaults()
	r.Format = "yaml"
	if err := r.Validate(); err == nil {
		t.Errorf("expected error for unsupported format")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
