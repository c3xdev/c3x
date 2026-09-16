package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/spf13/cobra"
)

// newPricingCmd assembles the `c3x pricing` family. The three actions
// give users direct control over the on-disk cache. Without this, the
// only way to recover from a stuck $0 row was `rm ~/.cache/c3x/cache.db`
// — fine for power users, not OK as a primary UX.
func newPricingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pricing",
		Short: "Manage the on-disk price cache.",
	}
	cmd.AddCommand(
		newPricingWhereCmd(),
		newPricingStatsCmd(),
		newPricingClearCmd(),
		newPricingSyncCmd(),
	)
	return cmd
}

// newPricingWhereCmd prints the resolved cache path. Useful when a user
// has set XDG_CACHE_HOME or --cache-path and wants to confirm where
// c3x actually writes.
func newPricingWhereCmd() *cobra.Command {
	var cachePath string
	cmd := &cobra.Command{
		Use:   "where",
		Short: "Print the on-disk cache path.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := resolveCachePath(cachePath)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
	cmd.Flags().StringVar(&cachePath, "cache-path", "", "override path (else platform default)")
	return cmd
}

// newPricingStatsCmd prints the row counts split by freshness so a
// user can see at a glance whether their cache is healthy.
func newPricingStatsCmd() *cobra.Command {
	var cachePath string
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show on-disk cache row counts (total, live, stale).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := resolveCachePath(cachePath)
			if err != nil {
				return err
			}
			if _, err := os.Stat(path); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(),
					"no cache file at %s yet — run `c3x estimate` to populate it\n", path)
				return nil
			}
			// Open the cache wrapping a no-op inner source. Stats only
			// reads from SQLite so the inner is irrelevant.
			cache, err := pricing.OpenDiskCache(path, pricing.NewStub())
			if err != nil {
				return fmt.Errorf("opening cache: %w", err)
			}
			defer func() { _ = cache.Close() }()

			stats, err := cache.Stats()
			if err != nil {
				return fmt.Errorf("reading stats: %w", err)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "path:  %s\n", path)
			fmt.Fprintf(out, "total: %d\n", stats.Total)
			fmt.Fprintf(out, "live:  %d\n", stats.Live)
			fmt.Fprintf(out, "stale: %d\n", stats.Stale)
			return nil
		},
	}
	cmd.Flags().StringVar(&cachePath, "cache-path", "", "override path (else platform default)")
	return cmd
}

// newPricingClearCmd empties the cache. The print-then-act ordering
// lets users CTRL-C if they ran it accidentally; the actual deletion
// happens in one atomic SQL statement.
func newPricingClearCmd() *cobra.Command {
	var cachePath string
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete every row from the on-disk cache.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := resolveCachePath(cachePath)
			if err != nil {
				return err
			}
			if _, err := os.Stat(path); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(),
					"no cache file at %s; nothing to clear\n", path)
				return nil
			}
			cache, err := pricing.OpenDiskCache(path, pricing.NewStub())
			if err != nil {
				return fmt.Errorf("opening cache: %w", err)
			}
			defer func() { _ = cache.Close() }()

			n, err := cache.Clear()
			if err != nil {
				return fmt.Errorf("clearing cache: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %d entries from %s\n", n, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&cachePath, "cache-path", "", "override path (else platform default)")
	return cmd
}

// resolveCachePath returns the configured cache path, defaulting to
// the XDG-aware platform location when the user hasn't overridden.
func resolveCachePath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	return config.UserCachePath()
}

// newPricingSyncCmd warms the on-disk cache with the full catalog price
// set so `c3x estimate --offline` returns real numbers. It enumerates
// every product the catalog can query (paginated, never a single bulk
// download) and stores each price keyed exactly as the engine asks for
// it. By default it warms each provider's reference region — enough for
// full coverage, since `--offline` falls back to the reference region
// for any other region. Pass --regions to warm extras for per-region
// accuracy.
func newPricingSyncCmd() *cobra.Command {
	var (
		cachePath       string
		pricingEndpoint string
		providers       []string
		regions         []string
		concurrency     int
	)
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Download the catalog price set for offline use.",
		Long: "Warm the on-disk price cache from the pricing API so " +
			"`c3x estimate --offline` works with real prices. Sync on a " +
			"connected machine, then copy the cache file (see `c3x pricing " +
			"where`) into an air-gapped network.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags := map[string]any{}
			if cachePath != "" {
				flags["cache_path"] = cachePath
			}
			if pricingEndpoint != "" {
				flags["pricing.endpoint"] = pricingEndpoint
			}
			resolved, err := config.Resolve(".", flags)
			if err != nil {
				return err
			}

			path := resolved.CachePath
			if path == "" {
				if path, err = config.UserCachePath(); err != nil {
					return fmt.Errorf("resolving default cache path: %w", err)
				}
			}

			reg, err := loadCatalogAuto(cmd.Context(), resolved)
			if err != nil {
				return fmt.Errorf("loading catalog: %w", err)
			}

			shapes := buildSyncShapes(reg, providers)
			if len(shapes) == 0 {
				return fmt.Errorf("no priced mappings to sync (check --providers)")
			}

			endpoint := resolved.PricingEndpoint
			if endpoint == "" {
				endpoint = pricing.DefaultEndpoint
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Syncing %d mappings from %s → %s\n", len(shapes), endpoint, path)

			start := time.Now()
			res, err := pricing.Sync(cmd.Context(), pricing.SyncOptions{
				Endpoint:    resolved.PricingEndpoint,
				Token:       resolved.PricingToken,
				CachePath:   path,
				Regions:     regions,
				Shapes:      shapes,
				Concurrency: concurrency,
				Progress: func(p pricing.SyncProgress) {
					fmt.Fprintf(out, "\r\033[K[%d/%d] %s/%s @ %s (+%d)",
						p.Done, p.Total, p.Provider, p.Service, p.Region, p.Entries)
				},
			})
			fmt.Fprintln(out)
			if err != nil {
				return fmt.Errorf("sync: %w", err)
			}
			fmt.Fprintf(out, "Done: %d prices · %d services · %d regions in %s\n",
				res.Entries, res.Services, res.Regions, time.Since(start).Round(time.Second))
			fmt.Fprintln(out, "Offline ready: c3x estimate --offline --path <dir>")
			return nil
		},
	}
	cmd.Flags().StringVar(&cachePath, "cache-path", "", "override path (else platform default)")
	cmd.Flags().StringVar(&pricingEndpoint, "pricing-endpoint", "", "override the GraphQL endpoint")
	cmd.Flags().StringSliceVar(&providers, "providers", nil, "limit to providers (aws,azure,gcp); default all")
	cmd.Flags().StringSliceVar(&regions, "regions", nil, "regions to warm; default each provider's reference region")
	cmd.Flags().IntVar(&concurrency, "concurrency", 0, "max concurrent service/region fetches (default 6)")
	return cmd
}

// buildSyncShapes derives the set of priced lookups from the catalog.
// Each mapping becomes a MappingShape; literal (`const`) filters are
// pinned, expr filters become dynamic keys resolved per product.
// Mappings with no upstream service (free/static-only) are skipped, and
// identical shapes across kinds are de-duplicated.
func buildSyncShapes(reg *catalog.Registry, providers []string) []pricing.MappingShape {
	want := map[string]bool{}
	for _, p := range providers {
		want[strings.TrimSpace(p)] = true
	}
	seen := map[string]bool{}
	var shapes []pricing.MappingShape
	for _, kind := range reg.Kinds() {
		def := reg.Get(kind)
		if def == nil {
			continue
		}
		if len(want) > 0 && !want[def.Provider] {
			continue
		}
		for _, m := range def.Mappings {
			if m.Service == "" {
				continue // free / static-rate mapping: no upstream lookup
			}
			var fixed []pricing.KV
			var keys []string
			for _, af := range m.AttributeFilters {
				if af.Literal != "" {
					fixed = append(fixed, pricing.KV{Key: af.Key, Value: af.Literal})
				} else if af.Key != "" {
					keys = append(keys, af.Key)
				}
			}
			shape := pricing.MappingShape{
				Provider:       def.Provider,
				Service:        m.Service,
				ProductFamily:  m.ProductFamily,
				FixedFilters:   fixed,
				FilterKeys:     keys,
				PurchaseOption: pricing.ResolvePurchaseOption(def.Provider, m.PurchaseOption),
				Unit:           m.Unit,
				RegionOverride: m.Region,
			}
			k := shapeDedupKey(shape)
			if seen[k] {
				continue
			}
			seen[k] = true
			shapes = append(shapes, shape)
		}
	}
	return shapes
}

func shapeDedupKey(s pricing.MappingShape) string {
	var b strings.Builder
	b.WriteString(s.Provider + "|" + s.Service + "|" + s.ProductFamily + "|" +
		s.PurchaseOption + "|" + s.Unit + "|" + s.RegionOverride + "|")
	for _, f := range s.FixedFilters {
		b.WriteString(f.Key + "=" + f.Value + ",")
	}
	b.WriteString("|")
	for _, k := range s.FilterKeys {
		b.WriteString(k + ",")
	}
	return b.String()
}
