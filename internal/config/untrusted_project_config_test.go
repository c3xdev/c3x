package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/config"
)

// A pull request from a fork can edit the scanned repository's .c3x.toml.
// In untrusted-input mode it must not be able to redirect traffic,
// credentials or file access; in trusted mode it keeps working as before.

func resolveWith(t *testing.T, projectTOML, userTOML string, flags map[string]any) config.Resolved {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if userTOML != "" {
		if err := os.MkdirAll(filepath.Join(xdg, "c3x"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(xdg, "c3x", "config.toml"), []byte(userTOML), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, ".c3x.toml"), []byte(projectTOML), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := config.Resolve(project, flags)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

const hostileProject = `
region = "eu-west-1"
offline = true
cache_path = "/tmp/pwned"
resources_path = "./evil-catalog"
usage_path = "/etc/passwd"
[pricing]
endpoint = "https://attacker.example/graphql"
token = "stolen"
`

func TestUntrustedModeFiltersProjectConfig(t *testing.T) {
	r := resolveWith(t, hostileProject, "", map[string]any{"no_remote_modules": true})
	def := config.Defaults()
	if r.PricingEndpoint != def.PricingEndpoint {
		t.Errorf("pricing.endpoint = %q; a PR must not redirect price lookups and the token", r.PricingEndpoint)
	}
	if r.PricingToken != "" {
		t.Errorf("pricing.token = %q; must not come from the project in untrusted mode", r.PricingToken)
	}
	if r.Offline {
		t.Error("offline must not come from the project: it swaps real prices for stubs")
	}
	if r.CachePath != def.CachePath {
		t.Errorf("cache_path=%q must not come from the project", r.CachePath)
	}
	if r.UsagePath != def.UsagePath {
		t.Errorf("usage_path = %q; a path outside the project must be refused", r.UsagePath)
	}
	if r.Region != "eu-west-1" {
		t.Errorf("region = %q; harmless keys still apply from the project", r.Region)
	}
}

func TestUntrustedModeKeepsUsagePathInsideProject(t *testing.T) {
	r := resolveWith(t, `usage_path = "usage/c3x-usage.yml"`, "", map[string]any{"no_remote_modules": true})
	if filepath.Base(filepath.Dir(r.UsagePath)) != "usage" || filepath.Base(r.UsagePath) != "c3x-usage.yml" {
		t.Errorf("usage_path = %q; a path inside the project is fine", r.UsagePath)
	}
}

// Untrusted mode set by the runner's own user config cannot be switched off
// by the project, which sits one layer above it.
func TestProjectCannotDisableUntrustedMode(t *testing.T) {
	r := resolveWith(t, "no_remote_modules = false\n"+hostileProject, "no_remote_modules = true\n", nil)
	if !r.NoRemoteModules {
		t.Error("a project must not turn untrusted-input mode off")
	}
	if r.PricingEndpoint != config.Defaults().PricingEndpoint {
		t.Error("with untrusted mode from user config, the project endpoint must still be ignored")
	}
}

// Trusted mode is unchanged: a self-hoster keeps pricing.endpoint in the repo.
func TestTrustedModeHonoursProjectConfig(t *testing.T) {
	r := resolveWith(t, `[pricing]
endpoint = "https://pricing.internal.example/graphql"
`, "", nil)
	if r.PricingEndpoint != "https://pricing.internal.example/graphql" {
		t.Errorf("pricing.endpoint = %q; trusted mode must honour the project config", r.PricingEndpoint)
	}
}
