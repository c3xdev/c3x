package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/spf13/viper"
)

// Resolve walks the 5-layer precedence stack and returns the merged
// configuration. Each layer overrides the previous one:
//
//  1. Defaults (in code)
//  2. User config file (~/.config/c3x/config.toml or platform equivalent)
//  3. Project config file (./.c3x.toml or the file projectDir resolves to)
//  4. Environment variables prefixed C3X_
//  5. CLI flag values (passed in via the `flags` map)
//
// flags is the snapshot of CLI flag values keyed by viper-style names
// (`region`, `format`, `pricing.endpoint`, …). Pass nil if there are no
// CLI overrides yet (useful in tests).
func Resolve(projectDir string, flags map[string]any) (Resolved, error) {
	v := viper.New()
	v.SetConfigType("toml")

	// Layer 1: defaults.
	d := Defaults()
	v.SetDefault("region", d.Region)
	v.SetDefault("currency", d.Currency.String())
	v.SetDefault("format", d.Format)
	v.SetDefault("offline", d.Offline)
	v.SetDefault("no_remote_modules", d.NoRemoteModules)
	v.SetDefault("no_cache", d.NoCache)
	v.SetDefault("cache_path", d.CachePath)
	v.SetDefault("usage_path", d.UsagePath)
	v.SetDefault("resources_path", d.ResourcesPath)
	v.SetDefault("budget", d.Budget)
	v.SetDefault("budget_delta", d.BudgetDelta)
	v.SetDefault("verbosity", d.Verbosity)
	v.SetDefault("pricing.endpoint", d.PricingEndpoint)

	// Layer 2: user config file (silent if missing).
	if userPath, err := UserConfigPath(); err == nil {
		if _, statErr := os.Stat(userPath); statErr == nil {
			v.SetConfigFile(userPath)
			if err := v.ReadInConfig(); err != nil {
				return Resolved{}, fmt.Errorf("user config %s: %w", userPath, err)
			}
		}
	}

	// Layer 3: project config file (silent if missing).
	projectPath := ProjectConfigPath(projectDir)
	if _, statErr := os.Stat(projectPath); statErr == nil {
		v.SetConfigFile(projectPath)
		if err := v.MergeInConfig(); err != nil {
			return Resolved{}, fmt.Errorf("project config %s: %w", projectPath, err)
		}
	}

	// Layer 4: environment variables, C3X_* with `_` replacing the dot
	// separator for nested keys.
	v.SetEnvPrefix("C3X")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Layer 5: CLI flag overrides. Each non-nil entry wins outright.
	for k, val := range flags {
		if val == nil {
			continue
		}
		v.Set(k, val)
	}

	// Materialise into the Resolved struct.
	currency, err := domain.ParseCurrency(v.GetString("currency"))
	if err != nil {
		return Resolved{}, fmt.Errorf("currency: %w", err)
	}

	out := Resolved{
		Region:          v.GetString("region"),
		Currency:        currency,
		Format:          v.GetString("format"),
		PricingEndpoint: v.GetString("pricing.endpoint"),
		Offline:         v.GetBool("offline"),
		NoRemoteModules: v.GetBool("no_remote_modules"),
		NoCache:         v.GetBool("no_cache"),
		CachePath:       v.GetString("cache_path"),
		UsagePath:       v.GetString("usage_path"),
		ResourcesPath:   v.GetString("resources_path"),
		Budget:          v.GetFloat64("budget"),
		BudgetDelta:     v.GetFloat64("budget_delta"),
		Verbosity:       v.GetInt("verbosity"),
	}
	if err := out.Validate(); err != nil {
		return Resolved{}, err
	}
	return out, nil
}

// ErrNoProjectDir is returned by helpers that need a project directory
// when the caller failed to supply one.
var ErrNoProjectDir = errors.New("project directory is empty")
