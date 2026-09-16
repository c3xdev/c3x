package main

import (
	"context"
	"fmt"
	"os"

	"github.com/c3xdev/c3x/internal/calculator"
	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/spf13/cobra"
)

// newDiffCmd compares a saved baseline (a JSON Estimate previously
// written by `c3x estimate --save-baseline`) against a freshly-parsed
// current state, then renders the Diff in the resolved format.
//
// In CI: `c3x diff --baseline base.json --budget-delta 50` runs on
// every PR. The exit code is non-zero if the delta exceeds the limit.
func newDiffCmd() *cobra.Command {
	var (
		path            string
		baselinePath    string
		format          string
		region          string
		varFiles        []string
		vars            []string
		offline         bool
		noCache         bool
		cachePath       string
		pricingEndpoint string
		budgetDelta     float64
	)

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare the current estimate against a saved baseline.",
		Long: `Loads a baseline JSON file (produced by previous
` + "`c3x estimate --save-baseline`" + `), runs a fresh estimate against
the supplied --path, and prints the delta resource-by-resource.

Together with --budget-delta this is the CI gate: PRs that increase
monthly spend by more than the configured amount fail the job.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if baselinePath == "" {
				return fmt.Errorf("--baseline is required")
			}
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
			if noCache {
				flags["no_cache"] = true
			}
			if cachePath != "" {
				flags["cache_path"] = cachePath
			}
			if pricingEndpoint != "" {
				flags["pricing.endpoint"] = pricingEndpoint
			}
			resolved, err := config.Resolve(projectDir, flags)
			if err != nil {
				return fmt.Errorf("resolving config: %w", err)
			}

			baseline, err := loadBaseline(baselinePath)
			if err != nil {
				return fmt.Errorf("loading baseline %s: %w", baselinePath, err)
			}

			current, err := computeCurrent(cmd.Context(), path, resolved, varFiles, vars)
			if err != nil {
				return err
			}

			diff := domain.ComputeDiff(baseline, current)
			f, err := render.ParseFormat(resolved.Format)
			if err != nil {
				return fmt.Errorf("format: %w", err)
			}
			out, err := render.RenderDiff(diff, f)
			if err != nil {
				return err
			}
			_, _ = cmd.OutOrStdout().Write([]byte(out))

			return enforceBudgetDelta(cmd, diff, budgetDelta)
		},
	}

	cmd.Flags().StringVar(&path, "path", ".", "Terraform input (directory, .tf, .hcl, or plan JSON)")
	cmd.Flags().StringVar(&baselinePath, "baseline", "", "path to the saved baseline JSON (required)")
	cmd.Flags().StringVar(&format, "format", "", "output format: text, markdown, json, junit, html, csv, sarif (overrides config)")
	cmd.Flags().StringVar(&region, "region", "", "default region when the IaC source doesn't declare one")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "tfvars files (repeatable)")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable override name=value (repeatable)")
	cmd.Flags().BoolVar(&offline, "offline", false, "use the offline pricing stub")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "bypass the on-disk price cache")
	cmd.Flags().StringVar(&cachePath, "cache-path", "", "override the cache file path")
	cmd.Flags().StringVar(&pricingEndpoint, "pricing-endpoint", "", "override the pricing GraphQL endpoint")
	cmd.Flags().Float64Var(&budgetDelta, "budget-delta", 0,
		"fail with exit code 1 when the project delta exceeds this monthly amount (0 disables the gate)")

	return cmd
}

// loadBaseline reads a baseline JSON file and decodes it into a
// domain.Estimate via the round-trippable contract owned by render.
func loadBaseline(path string) (domain.Estimate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return domain.Estimate{}, err
	}
	return render.DecodeEstimate(raw)
}

// computeCurrent re-runs the full parse → calculate pipeline for the
// `current` side of a diff. Mirrors the estimate command's flow so
// baseline and current are computed identically.
func computeCurrent(
	ctx context.Context,
	rawPath string,
	resolved config.Resolved,
	varFiles []string,
	rawVars []string,
) (domain.Estimate, error) {
	varMap, err := parseVarFlags(rawVars)
	if err != nil {
		return domain.Estimate{}, err
	}
	parsed, err := parser.Parse(rawPath, parser.Options{
		VarFiles: varFiles,
		Vars:     varMap,
		Offline:  resolved.Offline,
	})
	if err != nil {
		return domain.Estimate{}, fmt.Errorf("parsing %s: %w", rawPath, err)
	}
	reg, err := loadCatalogAuto(ctx, resolved)
	if err != nil {
		return domain.Estimate{}, fmt.Errorf("loading catalog: %w", err)
	}
	// Default cache path for both online (write-through) and offline
	// (read the warmed cache); only --no-cache opts out.
	pricePath := resolved.CachePath
	if pricePath == "" && !resolved.NoCache {
		def, err := config.UserCachePath()
		if err != nil {
			return domain.Estimate{}, fmt.Errorf("resolving default cache path: %w", err)
		}
		pricePath = def
	}
	prices, err := pricing.BuildChain(pricing.ChainOptions{
		Endpoint:  resolved.PricingEndpoint,
		Token:     resolved.PricingToken,
		CachePath: pricePath,
		Offline:   resolved.Offline,
		NoCache:   resolved.NoCache,
		Currency:  resolved.Currency,
	})
	if err != nil {
		return domain.Estimate{}, fmt.Errorf("building pricing chain: %w", err)
	}
	defer func() { _ = pricing.TryClose(prices) }()
	engine := calculator.New(calculator.Options{
		Registry:      reg,
		Prices:        prices,
		Currency:      resolved.Currency,
		DefaultRegion: coalesce(resolved.Region, "us-east-1"),
	})
	return engine.Estimate(ctx, parsed)
}

// enforceBudgetDelta exits non-zero if the delta exceeds the gate.
// The message goes to stderr so a `c3x diff --format json | jq`
// pipeline still produces valid JSON on stdout.
func enforceBudgetDelta(cmd *cobra.Command, d domain.Diff, limit float64) error {
	if limit <= 0 {
		return nil
	}
	delta, _ := d.TotalDelta.Float64()
	if delta <= limit {
		return nil
	}
	fmt.Fprintf(cmd.ErrOrStderr(),
		"c3x: delta exceeds gate — %s%+.2f/mo > %s%+.2f/mo\n",
		d.Currency.Symbol(), delta, d.Currency.Symbol(), limit)
	return errBudgetDeltaExceeded
}

// errBudgetDeltaExceeded is the sibling sentinel of errBudgetExceeded.
var errBudgetDeltaExceeded = fmt.Errorf("budget-delta exceeded")
