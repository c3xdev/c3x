package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/calculator"
	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/c3xdev/c3x/internal/recommend"
	"github.com/spf13/cobra"
)

// newRecommendCmd wires `c3x recommend`: walk the parsed
// configuration through every registered rule, score the proposed
// alternatives against the calculator, and surface savings.
//
// Output formats mirror estimate / diff for consistency across the
// CLI surface — text for terminals, markdown for PR comments, JSON
// for pipelines.
func newRecommendCmd() *cobra.Command {
	var (
		path            string
		format          string
		region          string
		varFiles        []string
		vars            []string
		usagePath       string
		whatIfs         []string
		offline         bool
		noCache         bool
		cachePath       string
		pricingEndpoint string
		pricingToken    string
	)
	cmd := &cobra.Command{
		Use:   "recommend",
		Short: "Suggest cost optimisations for the parsed infrastructure.",
		Long: `Walks every supported resource through the rule engine and prints
opportunities to reduce monthly spend. Each suggestion shows the
current cost, the cost under the proposed change, and the savings.

Rules include: gp2 → gp3 EBS migration, EBS right-sizing, Multi-AZ
RDS downgrade for non-prod, idle EIP audit, Azure burstable swap for
non-prod, GCP pd-standard → pd-balanced.`,
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
			if noCache {
				flags["no_cache"] = true
			}
			if cachePath != "" {
				flags["cache_path"] = cachePath
			}
			if pricingEndpoint != "" {
				flags["pricing.endpoint"] = pricingEndpoint
			}
			if pricingToken != "" {
				flags["pricing.token"] = pricingToken
			}
			if usagePath != "" {
				flags["usage_path"] = usagePath
			}
			resolved, err := config.Resolve(projectDir, flags)
			if err != nil {
				return fmt.Errorf("resolving config: %w", err)
			}

			varMap, err := parseVarFlags(vars)
			if err != nil {
				return err
			}
			parsed, err := parser.Parse(path, parserOptions(resolved, varFiles, varMap))
			if err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}
			if err := applyUsageAndWhatIf(cmd, parsed, resolved.UsagePath, whatIfs); err != nil {
				return err
			}

			reg, err := loadCatalogAuto(cmd.Context(), resolved)
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			// Default cache path for both online and offline (read the
			// warmed cache); only --no-cache opts out.
			pricePath := resolved.CachePath
			if pricePath == "" && !resolved.NoCache {
				def, err := config.UserCachePath()
				if err != nil {
					return fmt.Errorf("resolving default cache path: %w", err)
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
				return fmt.Errorf("building pricing chain: %w", err)
			}
			defer func() { _ = pricing.TryClose(prices) }()

			calc := calculator.New(calculator.Options{
				Registry:      reg,
				Prices:        prices,
				Currency:      resolved.Currency,
				DefaultRegion: coalesce(resolved.Region, "us-east-1"),
			})
			rules := append(append(append([]recommend.Rule{}, recommend.AWSRules()...),
				recommend.AzureRules()...), recommend.GCPRules()...)
			engine := recommend.New(calc, rules...)
			for _, tr := range recommend.AWSTreeRules() {
				engine.RegisterTreeRule(tr)
			}
			for _, tr := range recommend.AzureTreeRules() {
				engine.RegisterTreeRule(tr)
			}
			for _, tr := range recommend.GCPTreeRules() {
				engine.RegisterTreeRule(tr)
			}
			recs, err := engine.Recommend(cmd.Context(), parsed)
			if err != nil {
				return err
			}

			return writeRecommendations(cmd, recs, resolved.Currency, resolved.Format)
		},
	}
	cmd.Flags().StringVar(&path, "path", ".", "Terraform or OpenTofu input (directory, .tf, .tofu, .tf.json, .hcl, or plan JSON)")
	cmd.Flags().StringVar(&format, "format", "", "output format: text, markdown, json")
	cmd.Flags().StringVar(&region, "region", "", "default region when the IaC source doesn't declare one")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "tfvars files (repeatable)")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable override name=value (repeatable)")
	cmd.Flags().StringVar(&usagePath, "usage", "", "path to a c3x-usage.yml file")
	cmd.Flags().StringArrayVar(&whatIfs, "what-if", nil, "attribute override kind.name.attr=value (repeatable)")
	cmd.Flags().BoolVar(&offline, "offline", false, "use the offline pricing stub")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "bypass the on-disk price cache")
	cmd.Flags().StringVar(&cachePath, "cache-path", "", "override the cache file path")
	cmd.Flags().StringVar(&pricingEndpoint, "pricing-endpoint", "", "override the GraphQL endpoint")
	cmd.Flags().StringVar(&pricingToken, "pricing-token", "", "bearer token for a self-hosted pricing API (default: $C3X_PRICING_TOKEN)")
	return cmd
}

// writeRecommendations renders recommendations locally so the
// recommend command is fully self-contained.
func writeRecommendations(cmd *cobra.Command, recs []recommend.Recommendation, cur domain.Currency, formatName string) error {
	if formatName == "" {
		formatName = "text"
	}
	out := cmd.OutOrStdout()
	switch strings.ToLower(formatName) {
	case "json":
		view := make([]map[string]any, 0, len(recs))
		for i := range recs {
			r := &recs[i]
			view = append(view, map[string]any{
				"resource":       r.Resource.Label(),
				"category":       r.Category,
				"title":          r.Title,
				"description":    r.Description,
				"current_cost":   r.CurrentCost.String(),
				"suggested_cost": r.SuggestedCost.String(),
				"savings":        r.Savings.String(),
				"currency":       cur.String(),
			})
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(view)
	case "markdown", "md":
		if len(recs) == 0 {
			fmt.Fprintln(out, "## c3x recommend\n\n_No optimisations found._")
			return nil
		}
		fmt.Fprintln(out, "## c3x recommend")
		for i := range recs {
			r := &recs[i]
			fmt.Fprintf(out, "\n### `%s` — %s\n\n", r.Resource.Label(), r.Title)
			fmt.Fprintf(out, "_%s_\n\n", r.Description)
			fmt.Fprintf(out, "| Current | Suggested | Savings |\n|---:|---:|---:|\n")
			fmt.Fprintf(out, "| %s%s/mo | %s%s/mo | **%s%s/mo** |\n",
				cur.Symbol(), r.CurrentCost,
				cur.Symbol(), r.SuggestedCost,
				cur.Symbol(), r.Savings)
		}
		return nil
	}
	// Text (default).
	if len(recs) == 0 {
		fmt.Fprintln(out, "c3x: no optimisations found.")
		return nil
	}
	fmt.Fprintln(out, "── c3x recommend ─────────────────────────────────────────────")
	for i := range recs {
		r := &recs[i]
		fmt.Fprintf(out, "\n  %s   save %s%s/mo (%s%s → %s%s)\n",
			r.Resource.Label(),
			cur.Symbol(), r.Savings,
			cur.Symbol(), r.CurrentCost,
			cur.Symbol(), r.SuggestedCost)
		fmt.Fprintf(out, "    %s\n", r.Title)
		fmt.Fprintf(out, "    %s\n", r.Description)
	}
	return nil
}
