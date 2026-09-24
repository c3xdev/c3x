package main

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/spf13/cobra"
)

// newTopCmd wires `c3x top`: rank the resources of an estimate by
// monthly cost, highest first.
//
// The motivating use case (#64) is scheduled teardown of expensive
// development infrastructure: `--format targets` emits the ranking as
// `-target=` flags, so CI can feed the N costliest resources straight
// into `tofu destroy`. c3x only ever produces the list; it never
// destroys anything itself.
func newTopCmd() *cobra.Command {
	var (
		path            string
		format          string
		limit           int
		region          string
		varFiles        []string
		vars            []string
		offline         bool
		noCache         bool
		cachePath       string
		pricingEndpoint string
		pricingToken    string
	)

	cmd := &cobra.Command{
		Use:   "top",
		Short: "List the most expensive resources, ranked by monthly cost.",
		Long: `Ranks the resources in an estimate by monthly cost, highest first.

Formats:
  text     ranked table with each resource's share of the total (default)
  targets  ` + "`-target=<address>`" + ` flags, one per line
  json     structured ranking for tooling

The targets format is built for scheduled teardown of costly development
environments:

  tofu destroy $(c3x top --limit 5 --format targets)

Addresses are canonical Terraform addresses, so resources inside modules
target correctly. Free resources are never listed: destroying them saves
nothing.

c3x only prints the list, it never destroys anything. Review it before
piping it into a destroy: ` + "`-target`" + ` is a blunt instrument, and
tearing down a resource that others depend on can break the rest of the
environment.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if limit < 0 {
				return fmt.Errorf("--limit must be zero or positive (0 lists every priced resource)")
			}
			projectDir, err := resolveProjectDir(path)
			if err != nil {
				return err
			}

			flags := map[string]any{}
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
			resolved, err := config.Resolve(projectDir, flags)
			if err != nil {
				return fmt.Errorf("resolving config: %w", err)
			}

			est, err := computeCurrent(cmd.Context(), path, resolved, varFiles, vars)
			if err != nil {
				return err
			}

			var out string
			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "text":
				out = render.RenderTopText(est, limit)
			case "targets":
				out = render.RenderTopTargets(est, limit)
			case "json":
				out, err = render.RenderTopJSON(est, limit)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported --format %q for top (want text, targets, or json)", format)
			}
			_, _ = cmd.OutOrStdout().Write([]byte(out))
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "path", ".", "Terraform or OpenTofu input (directory, .tf, .tofu, .hcl, or plan JSON)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text, targets, json")
	cmd.Flags().IntVar(&limit, "limit", 10, "how many resources to list (0 lists every priced resource)")
	cmd.Flags().StringVar(&region, "region", "", "default region when the IaC source doesn't declare one")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "tfvars files (repeatable)")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable override name=value (repeatable)")
	cmd.Flags().BoolVar(&offline, "offline", false, "use the offline pricing stub")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "bypass the on-disk price cache")
	cmd.Flags().StringVar(&cachePath, "cache-path", "", "override the cache file path")
	cmd.Flags().StringVar(&pricingEndpoint, "pricing-endpoint", "", "override the pricing GraphQL endpoint")
	cmd.Flags().StringVar(&pricingToken, "pricing-token", "", "bearer token for a self-hosted pricing API (default: $C3X_PRICING_TOKEN)")

	return cmd
}
