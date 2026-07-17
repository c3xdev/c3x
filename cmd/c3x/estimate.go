package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c3xdev/c3x/internal/calculator"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/c3xdev/c3x/internal/usage"
	"github.com/c3xdev/c3x/internal/whatif"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

// newEstimateCmd wires the user-facing estimate command. The pipeline:
//
//	parser.Parse(path)  → []domain.Resource
//	calculator.Engine.Estimate(resources)  → domain.Estimate
//	render.Render(estimate, format)  → text / markdown / JSON / ...
//
// Renderers cover text, markdown, JSON, JUnit, HTML, CSV, and SARIF.
func newEstimateCmd() *cobra.Command {
	var (
		path            string
		format          string
		region          string
		varFiles        []string
		vars            []string
		usagePath       string
		whatIfs         []string
		offline         bool
		noRemoteModules bool
		noCache         bool
		cachePath       string
		pricingEndpoint string
		saveBaseline    string
		budget          float64
		inlineDemo      bool
		currency        string
		showSkipped     bool
	)

	cmd := &cobra.Command{
		Use:   "estimate",
		Short: "Estimate the monthly cost of infrastructure at the given path.",
		Long: `Reads Terraform (.tf, plan JSON) at the target path, prices every
supported resource against pricing.c3x.dev, and prints a per-resource
breakdown.

Variables can be supplied via .tfvars files (auto-loaded from the
project directory), --var-file flags, or --var name=value pairs. The
precedence matches Terraform's: defaults < auto.tfvars < --var-file <
--var.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			projectDir, err := resolveProjectDir(path)
			if err != nil {
				return err
			}

			flags := map[string]any{}
			if format != "" {
				flags["format"] = format
			}
			if region != "" {
				flags["region"] = region
			}
			if offline {
				flags["offline"] = true
			}
			if noRemoteModules {
				flags["no_remote_modules"] = true
			}
			if noCache {
				flags["no_cache"] = true
			}
			if cachePath != "" {
				flags["cache_path"] = cachePath
			}
			if pricingEndpoint != "" {
				flags["pricing.endpoint"] = pricingEndpoint
			}
			if currency != "" {
				flags["currency"] = currency
			}

			resolved, err := config.Resolve(projectDir, flags)
			if err != nil {
				return fmt.Errorf("resolving config: %w", err)
			}

			if inlineDemo {
				return runInlineDemo(cmd, resolved, budget)
			}

			_ = projectDir // resolved.* already carries the project config
			return runEstimate(cmd, path, resolved, varFiles, vars, usagePath, whatIfs, saveBaseline, budget, showSkipped)
		},
	}

	cmd.Flags().StringVar(&path, "path", ".", "Terraform input (directory, .tf, .hcl, or plan JSON)")
	cmd.Flags().StringVar(&format, "format", "", "output format: text, markdown, json, junit, html, csv, sarif (overrides config)")
	cmd.Flags().StringVar(&region, "region", "", "default region when the IaC source doesn't declare one")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil,
		"path to a .tfvars or .tfvars.json file; applied after auto-tfvars (repeatable)")
	cmd.Flags().StringArrayVar(&vars, "var", nil,
		"variable override `name=value` (repeatable; HCL or bare string accepted)")
	cmd.Flags().BoolVar(&offline, "offline", false,
		"skip the network and use the offline pricing stub (subtotals will be $0 for most resources)")
	cmd.Flags().BoolVar(&noRemoteModules, "no-remote-modules", false,
		"disable network module fetching (Registry/Git/HTTP) while keeping live pricing; use when parsing untrusted Terraform")
	cmd.Flags().BoolVar(&noCache, "no-cache", false,
		"bypass the on-disk price cache (every lookup goes to pricing.c3x.dev)")
	cmd.Flags().StringVar(&cachePath, "cache-path", "",
		"SQLite file used by the on-disk price cache (default: platform XDG cache dir)")
	cmd.Flags().StringVar(&pricingEndpoint, "pricing-endpoint", "",
		"override the GraphQL endpoint (default: https://pricing.c3x.dev/graphql)")

	cmd.Flags().StringVar(&usagePath, "usage", "",
		"path to a c3x-usage.yml file with runtime usage quantities (monthly_requests, monthly_storage_gb, etc.)")
	cmd.Flags().StringArrayVar(&whatIfs, "what-if", nil,
		"override an attribute: `kind.name.attr=value` (repeatable; bool/int/float/string coercion)")
	cmd.Flags().StringVar(&saveBaseline, "save-baseline", "",
		"after the estimate, write the JSON representation to this path for use as a `c3x diff` baseline")
	cmd.Flags().Float64Var(&budget, "budget", 0,
		"fail with exit code 1 when the project total exceeds this monthly budget (0 disables the gate)")
	cmd.Flags().StringVar(&currency, "currency", "",
		"display currency (USD, EUR, GBP, JPY, CAD, AUD, …); USD-priced rates are converted via Frankfurter")
	cmd.Flags().BoolVar(&showSkipped, "show-skipped", false,
		"after the breakdown, list resources we parsed but couldn't price, with the reason")

	cmd.Flags().BoolVar(&inlineDemo, "inline-demo", false,
		"drive the calculator against a hand-crafted aws_instance (dev surface)")
	_ = cmd.Flags().MarkHidden("inline-demo")

	return cmd
}

// runEstimate is the happy path: parse the input, run the calculator,
// and print a breakdown.
func runEstimate(
	cmd *cobra.Command,
	rawPath string,
	resolved config.Resolved,
	varFiles []string,
	rawVars []string,
	usagePath string,
	whatIfs []string,
	saveBaseline string,
	budget float64,
	showSkipped bool,
) error {
	varMap, err := parseVarFlags(rawVars)
	if err != nil {
		return err
	}
	parsed, err := parser.Parse(rawPath, parser.Options{
		VarFiles: varFiles,
		Vars:     varMap,
		// Disable remote module fetching when pricing is offline OR when the
		// caller opted into untrusted-input mode (--no-remote-modules). The
		// pricing chain below still honors resolved.Offline independently.
		Offline: resolved.Offline || resolved.NoRemoteModules,
	})
	if err != nil {
		return fmt.Errorf("parsing %s: %w", rawPath, err)
	}

	if err := applyUsageAndWhatIf(cmd, parsed, usagePath, whatIfs); err != nil {
		return err
	}

	reg, err := loadCatalogAuto(cmd.Context(), resolved)
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}

	// Resolve the default cache path for both online (write-through
	// cache) and offline (read the warmed cache). Only --no-cache opts
	// out entirely.
	cachePath := resolved.CachePath
	if cachePath == "" && !resolved.NoCache {
		defaultPath, perr := config.UserCachePath()
		if perr != nil {
			return fmt.Errorf("resolving default cache path: %w", perr)
		}
		cachePath = defaultPath
	}

	prices, err := pricing.BuildChain(pricing.ChainOptions{
		Endpoint:  resolved.PricingEndpoint,
		CachePath: cachePath,
		Offline:   resolved.Offline,
		NoCache:   resolved.NoCache,
		Currency:  resolved.Currency,
	})
	if err != nil {
		return fmt.Errorf("building pricing chain: %w", err)
	}
	defer func() { _ = pricing.TryClose(prices) }()

	engine := calculator.New(calculator.Options{
		Registry:      reg,
		Prices:        prices,
		Currency:      resolved.Currency,
		DefaultRegion: coalesce(resolved.Region, "us-east-1"),
	})
	// Use the cobra command's context so Ctrl+C cancels the
	// in-flight estimate (pricing HTTP calls, expression eval) at
	// the next checkpoint instead of leaving the CLI unresponsive.
	est, err := engine.Estimate(cmd.Context(), parsed)
	if err != nil {
		return fmt.Errorf("estimating: %w", err)
	}

	if err := writeRendered(cmd, est, resolved.Format); err != nil {
		return err
	}

	if len(est.Skipped) > 0 {
		// Print to stderr so machine-readable formats (--format json,
		// --format csv) on stdout stay parseable.
		if showSkipped {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"\n%d resource(s) were detected but not priced:\n", len(est.Skipped))
			for _, s := range est.Skipped {
				fmt.Fprintf(cmd.ErrOrStderr(), "  - %s.%s: %s\n",
					s.Resource.Kind, s.Resource.Name, s.Reason)
			}
		} else {
			// Warn by default: a resource that couldn't be priced must
			// not be silently dropped, or users assume it's free.
			verb := "resources were"
			if len(est.Skipped) == 1 {
				verb = "resource was"
			}
			fmt.Fprintf(cmd.ErrOrStderr(),
				"\nwarning: %d %s detected but not priced and excluded from the "+
					"total; re-run with --show-skipped for details.\n",
				len(est.Skipped), verb)
		}
	}

	if saveBaseline != "" {
		if err := writeBaseline(est, saveBaseline); err != nil {
			return fmt.Errorf("--save-baseline: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "baseline saved to %s\n", saveBaseline)
	}

	return enforceBudget(cmd, est, budget)
}

// writeBaseline persists the estimate to disk as JSON, the input
// format `c3x diff --baseline` expects. We always use the JSON
// renderer (regardless of resolved.Format) because the JSON view
// is the file-format contract.
func writeBaseline(est domain.Estimate, path string) error {
	raw, err := render.RenderJSON(est)
	if err != nil {
		return fmt.Errorf("rendering baseline JSON: %w", err)
	}
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// enforceBudget exits non-zero if the configured budget is breached.
// The diagnostic goes to stderr so a `c3x estimate --format json |
// jq` pipeline still produces valid JSON on stdout.
func enforceBudget(cmd *cobra.Command, est domain.Estimate, budget float64) error {
	if budget <= 0 {
		return nil
	}
	total, _ := est.ProjectTotal.Float64()
	if total <= budget {
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(),
		"c3x: budget exceeded — %s%.2f/mo > %s%.2f/mo\n",
		est.Currency.Symbol(), total, est.Currency.Symbol(), budget)
	return errBudgetExceeded
}

// errBudgetExceeded is a sentinel so cmd/c3x can set the appropriate
// exit code without duplicating the message printed by enforceBudget.
var errBudgetExceeded = fmt.Errorf("budget exceeded")

// applyUsageAndWhatIf is the shared "after parse, before calculate"
// hook used by estimate / diff / recommend. Usage file applies first
// (runtime quantities), then --what-if overrides (CLI wins).
// Unmatched entries in either log a warning to stderr.
func applyUsageAndWhatIf(cmd *cobra.Command, resources []domain.Resource, usagePath string, whatIfs []string) error {
	if usagePath != "" {
		f, err := usage.Load(usagePath)
		if err != nil {
			return fmt.Errorf("--usage: %w", err)
		}
		if unmatched := usage.Apply(resources, f); len(unmatched) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"c3x: warning: %d usage entries did not match any resource (%v)\n",
				len(unmatched), unmatched)
		}
	}
	if len(whatIfs) > 0 {
		ovs, err := whatif.Parse(whatIfs)
		if err != nil {
			return fmt.Errorf("--what-if: %w", err)
		}
		if unmatched := whatif.Apply(resources, ovs); len(unmatched) > 0 {
			for _, o := range unmatched {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"c3x: warning: --what-if %s.%s.%s=... did not match any resource\n",
					o.Kind, o.Name, o.Attr)
			}
		}
	}
	return nil
}

// writeRendered routes the estimate through the appropriate renderer
// and writes to the command's stdout. Returning the render error lets
// the CLI surface a clean diagnostic if marshalling fails.
func writeRendered(cmd *cobra.Command, est domain.Estimate, formatName string) error {
	f, err := render.ParseFormat(formatName)
	if err != nil {
		return fmt.Errorf("format: %w", err)
	}
	out, err := render.Render(est, f)
	if err != nil {
		return fmt.Errorf("rendering: %w", err)
	}
	_, _ = cmd.OutOrStdout().Write([]byte(out))
	return nil
}

// parseVarFlags turns `--var name=value` repeats into a map.
func parseVarFlags(raw []string) (map[string]string, error) {
	out := map[string]string{}
	for _, kv := range raw {
		eq := strings.IndexRune(kv, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("--var expects NAME=VALUE, got %q", kv)
		}
		out[strings.TrimSpace(kv[:eq])] = kv[eq+1:]
	}
	return out, nil
}

// runInlineDemo runs a self-contained smoke test with a stubbed
// price, independent of the parser path. Kept for dev iteration.
func runInlineDemo(cmd *cobra.Command, resolved config.Resolved, budget float64) error {
	reg, err := catalog.Load()
	if err != nil {
		return fmt.Errorf("loading catalog: %w", err)
	}
	stub := pricing.NewStub()
	stub.Set(pricing.Query{
		Provider:       "aws",
		Service:        "AmazonEC2",
		ProductFamily:  "Compute Instance",
		Region:         "us-east-1",
		PurchaseOption: "on_demand",
		AttributeFilters: []pricing.KV{
			{Key: "instanceType", Value: "m5.xlarge"},
			{Key: "operatingSystem", Value: "Linux"},
			{Key: "preInstalledSw", Value: "NA"},
			{Key: "tenancy", Value: "Shared"},
			{Key: "capacitystatus", Value: "Used"},
		},
	}, decimal.RequireFromString("0.192"))

	engine := calculator.New(calculator.Options{
		Registry:      reg,
		Prices:        stub,
		Currency:      resolved.Currency,
		DefaultRegion: coalesce(resolved.Region, "us-east-1"),
	})
	region := "us-east-1"
	res := domain.Resource{
		Ref:    domain.Reference{Kind: "aws_instance", Name: "demo"},
		Region: &region,
		Attributes: map[string]any{
			"instance_type": "m5.xlarge",
		},
	}
	est, err := engine.Estimate(cmd.Context(), []domain.Resource{res})
	if err != nil {
		return fmt.Errorf("estimating: %w", err)
	}
	if err := writeRendered(cmd, est, resolved.Format); err != nil {
		return err
	}
	return enforceBudget(cmd, est, budget)
}

// resolveProjectDir takes a --path that may be a file or directory and
// returns the directory config.Resolve should read .c3x.toml from.
func resolveProjectDir(path string) (string, error) {
	if path == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving %q: %w", path, err)
	}
	stat, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", abs, err)
	}
	if stat.IsDir() {
		return abs, nil
	}
	return filepath.Dir(abs), nil
}

func coalesce(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
