package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
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
	v.SetDefault("pricing.token", d.PricingToken)

	// Layer 2: user config file (silent if missing).
	if userPath, err := UserConfigPath(); err == nil {
		if _, statErr := os.Stat(userPath); statErr == nil {
			v.SetConfigFile(userPath)
			if err := v.ReadInConfig(); err != nil {
				return Resolved{}, fmt.Errorf("user config %s: %w", userPath, err)
			}
		}
	}

	// allow_file_functions must not come from the project config, which
	// lives in the scanned directory and so is attacker-controlled on a
	// pull request. Capture it from defaults + user config now; env and
	// flags are applied on top below, and the project layer is skipped.
	allowFileFunctions := v.GetBool("allow_file_functions")

	// Is the runner treating this input as untrusted? Decided only from
	// sources outside the scanned directory — defaults and user config so
	// far, then the environment and flags — because the project config is
	// part of the input being judged.
	untrusted := v.GetBool("no_remote_modules")
	if env, ok := os.LookupEnv("C3X_NO_REMOTE_MODULES"); ok {
		if b, err := strconv.ParseBool(env); err == nil {
			untrusted = b
		}
	}
	if f, ok := flags["no_remote_modules"].(bool); ok {
		untrusted = f
	}

	// Layer 3: project config file (silent if missing), filtered.
	projectPath := ProjectConfigPath(projectDir)
	if _, statErr := os.Stat(projectPath); statErr == nil {
		pv := viper.New()
		pv.SetConfigType("toml")
		pv.SetConfigFile(projectPath)
		if err := pv.ReadInConfig(); err != nil {
			return Resolved{}, fmt.Errorf("project config %s: %w", projectPath, err)
		}
		settings, ignored := projectSettings(pv, projectDir, untrusted)
		if err := v.MergeConfigMap(settings); err != nil {
			return Resolved{}, fmt.Errorf("project config %s: %w", projectPath, err)
		}
		for _, key := range ignored {
			slog.Warn("ignoring a setting in the project's .c3x.toml; set it with a flag, "+
				"a C3X_ environment variable or your user config instead",
				"key", key, "file", projectPath, "untrusted_input", untrusted)
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

	if env, ok := os.LookupEnv("C3X_ALLOW_FILE_FUNCTIONS"); ok {
		if b, err := strconv.ParseBool(env); err == nil {
			allowFileFunctions = b
		}
	}
	if f, ok := flags["allow_file_functions"].(bool); ok {
		allowFileFunctions = f
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
		PricingToken:    v.GetString("pricing.token"),
		Offline:         v.GetBool("offline"),
		NoRemoteModules: v.GetBool("no_remote_modules"),
		// Untrusted-input mode wins over any opt-in.
		AllowFileFunctions: allowFileFunctions && !v.GetBool("no_remote_modules"),
		NoCache:            v.GetBool("no_cache"),
		CachePath:          v.GetString("cache_path"),
		UsagePath:          v.GetString("usage_path"),
		ResourcesPath:      v.GetString("resources_path"),
		Budget:             v.GetFloat64("budget"),
		BudgetDelta:        v.GetFloat64("budget_delta"),
		Verbosity:          v.GetInt("verbosity"),
	}
	if err := out.Validate(); err != nil {
		return Resolved{}, err
	}
	return out, nil
}

// ErrNoProjectDir is returned by helpers that need a project directory
// when the caller failed to supply one.
var ErrNoProjectDir = errors.New("project directory is empty")

// projectSafeKeys may come from a project's .c3x.toml even when the input
// is untrusted: they shape how an estimate is presented or gated, and
// cannot redirect network traffic, credentials or file access.
var projectSafeKeys = map[string]bool{
	"region": true, "currency": true, "format": true, "verbosity": true,
	"budget": true, "budget_delta": true, "no_cache": true, "usage_path": true,
}

// projectSettings returns the project config as a nested map to merge,
// and the keys it dropped.
//
// allow_file_functions is always dropped: it would let a pull request
// enable file reads on itself. In untrusted-input mode the file is
// limited to projectSafeKeys, because a pull request from a fork can
// edit it: pricing.endpoint would send the pricing token and every price
// lookup to a server of the attacker's choosing (and could fake prices
// to pass a budget gate), cache_path and resources_path point c3x at
// arbitrary paths, and offline swaps real prices for stubs. usage_path is
// kept only when it stays inside the project, since a parse error on an
// arbitrary file would quote its contents. no_remote_modules may only be
// turned on, never off.
func projectSettings(pv *viper.Viper, projectDir string, untrusted bool) (map[string]any, []string) {
	out := map[string]any{}
	var ignored []string
	for _, key := range pv.AllKeys() {
		val := pv.Get(key)
		keep := key != "allow_file_functions"
		if keep && untrusted {
			switch key {
			case "no_remote_modules":
				keep = pv.GetBool(key)
			case "usage_path":
				keep = insideDir(projectDir, pv.GetString(key))
			default:
				keep = projectSafeKeys[key]
			}
		}
		if !keep {
			ignored = append(ignored, key)
			continue
		}
		setNested(out, strings.Split(key, "."), val)
	}
	return out, ignored
}

func setNested(m map[string]any, path []string, val any) {
	for _, k := range path[:len(path)-1] {
		next, ok := m[k].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[k] = next
		}
		m = next
	}
	m[path[len(path)-1]] = val
}

// insideDir reports whether p, taken relative to dir when not absolute,
// stays within dir.
func insideDir(dir, p string) bool {
	if p == "" {
		return true
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(absDir, p)
	}
	rel, err := filepath.Rel(absDir, filepath.Clean(p))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
