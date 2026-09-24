package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/config"
)

// budget, budget_delta and usage_path in .c3x.toml were documented but
// never read: the commands only looked at their flags.

func TestProjectConfigSetsGates(t *testing.T) {
	r := resolveWith(t, "budget = 1000.0\nbudget_delta = 50\n", "", nil)
	if r.Budget != 1000 || r.BudgetDelta != 50 {
		t.Errorf("budget=%v budget_delta=%v, want 1000 and 50 from .c3x.toml", r.Budget, r.BudgetDelta)
	}
}

func TestFlagOverridesProjectBudget(t *testing.T) {
	// An explicit --budget 0 turns the project's gate off.
	r := resolveWith(t, "budget = 1000.0\n", "", map[string]any{"budget": 0.0})
	if r.Budget != 0 {
		t.Errorf("budget = %v, want the flag's 0 to win", r.Budget)
	}
}

func TestProjectUsagePathIsRelativeToProject(t *testing.T) {
	r := resolveWith(t, `usage_path = "c3x-usage.yml"`+"\n", "", nil)
	if !filepath.IsAbs(r.UsagePath) || filepath.Base(r.UsagePath) != "c3x-usage.yml" {
		t.Fatalf("usage_path = %q, want an absolute path inside the project", r.UsagePath)
	}
}

func TestFlagUsagePathIsLeftAlone(t *testing.T) {
	r := resolveWith(t, "", "", map[string]any{"usage_path": "rel/usage.yml"})
	if r.UsagePath != "rel/usage.yml" {
		t.Errorf("usage_path = %q; a flag path stays relative to the working directory", r.UsagePath)
	}
}

func TestNegativeBudgetDeltaRejected(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, ".c3x.toml"), []byte("budget_delta = -5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Resolve(project, nil); err == nil {
		t.Error("a negative budget_delta must be rejected")
	}
}
