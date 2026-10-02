package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/parser"
	"github.com/c3xdev/c3x/internal/usagesync"
	awsusage "github.com/c3xdev/c3x/internal/usagesync/aws"
	"github.com/spf13/cobra"
)

// newUsageCmd assembles the `c3x usage` family.
func newUsageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Fill usage files from cloud metrics.",
	}
	cmd.AddCommand(newUsageSyncCmd())
	return cmd
}

// newUsageSource builds the metric source for a provider. It is a variable
// so tests can supply a fake and run without cloud credentials.
var newUsageSource = func(ctx context.Context, provider string) (usagesync.Source, error) {
	switch provider {
	case "aws":
		return awsusage.New(ctx)
	default:
		return nil, fmt.Errorf("unsupported provider %q (supported: aws)", provider)
	}
}

func newUsageSyncCmd() *cobra.Command {
	var (
		path, statePath, provider, out, region string
		days                                   int
		strict                                 bool
		varFiles, rawVars                      []string
	)
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Write measured usage (quantities) from your cloud account's metrics.",
		Long: `Reads the resources in --path, looks up what each one actually used over
the last --days in your cloud account, and writes the result to a generated
usage file (c3x-usage.synced.yml). Every command that prices reads it.

This is not 'c3x pricing sync', which warms the local cache of prices
(rates). usage sync measures quantities: GB stored, requests served.

The generated file is replaced whole on every run and is never edited by
hand. Your hand-written usage file (usage_path) is never read or written
here, and wins over the generated one per resource and per key.

Only this command uses cloud credentials, and only to make read-only
calls. They come from the cloud SDK's default chain (environment, shared
config, SSO, instance or task role). It refuses to run in untrusted-input
mode (--no-remote-modules / C3X_NO_REMOTE_MODULES), so a pull request from
a fork can never trigger calls made with the runner's credentials.

To find a resource in the cloud its real name is needed. Pass the
Terraform state (terraform show -json > state.json) with --state; without
it a literal name in the configuration is used. A resource whose name or
region cannot be established is skipped and listed, never guessed.

AWS: S3 bucket storage (standard_storage_gb) from CloudWatch
BucketSizeBytes. Needs cloudwatch:GetMetricData; GetMetricData is billed
per metric requested, a small amount for a project's buckets.`,
		Example: `  terraform show -json > state.json
  c3x usage sync --state state.json
  c3x estimate`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags := map[string]any{}
			if region != "" {
				flags["region"] = region
			}
			resolved, err := config.Resolve(".", flags)
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			if resolved.NoRemoteModules {
				return errors.New("usage sync refuses to run in untrusted-input mode " +
					"(--no-remote-modules / C3X_NO_REMOTE_MODULES): it would use this machine's cloud credentials")
			}
			if days < 1 {
				return fmt.Errorf("--days must be at least 1, got %d", days)
			}
			outPath, err := usageSyncOutPath(out, resolved)
			if err != nil {
				return err
			}

			varMap, err := parseVarFlags(rawVars)
			if err != nil {
				return err
			}
			opts := parserOptions(resolved, varFiles, varMap)
			resources, err := parser.Parse(path, opts)
			if err != nil {
				return fmt.Errorf("parsing %s: %w", path, err)
			}
			var state []domain.Resource
			if statePath != "" {
				if state, err = parser.ParseState(statePath, opts); err != nil {
					return fmt.Errorf("parsing state %s: %w", statePath, err)
				}
			}

			source, err := newUsageSource(cmd.Context(), provider)
			if err != nil {
				return err
			}
			targets, problems := usagesync.Resolve(resources, state, statePath != "", resolved.Region, source.Emits())

			snap, err := usagesync.Run(cmd.Context(), usagesync.Options{
				Source:     source,
				Targets:    targets,
				Problems:   problems,
				WindowDays: days,
			})
			if err != nil {
				return fmt.Errorf("sync: %w", err)
			}
			if err := snap.Write(outPath); err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			if len(targets)+len(problems) == 0 {
				fmt.Fprintf(w, "No resources of a supported kind (%s) found in %s.\n",
					strings.Join(supportedKinds(source), ", "), path)
			}
			fmt.Fprintf(w, "Synced %d resource(s) from %s over %d days → %s\n",
				len(snap.ResourceUsage), source.Provider(), days, outPath)
			for _, addr := range sortedErrorKeys(snap.Errors) {
				fmt.Fprintf(w, "  skipped %s: %s\n", addr, snap.Errors[addr])
			}
			if n := len(snap.Errors); n > 0 && strict {
				return fmt.Errorf("%d resource(s) could not be synced (--strict)", n)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", ".", "Terraform directory, file or plan JSON to read resources from")
	cmd.Flags().StringVar(&statePath, "state", "", "Terraform state JSON (terraform show -json) to read real resource names from")
	cmd.Flags().StringVar(&provider, "provider", "aws", "cloud provider to read metrics from (aws)")
	cmd.Flags().IntVar(&days, "days", 30, "length of the window to measure usage over, in days")
	cmd.Flags().StringVar(&out, "out", "", "file to write; default synced_usage_path, else c3x-usage.synced.yml")
	cmd.Flags().StringVar(&region, "region", "", "region for resources that have none of their own")
	cmd.Flags().BoolVar(&strict, "strict", false, "exit non-zero if any resource could not be synced")
	cmd.Flags().StringSliceVar(&varFiles, "var-file", nil, "Terraform variable file (repeatable)")
	cmd.Flags().StringSliceVar(&rawVars, "var", nil, "Terraform variable name=value (repeatable)")
	return cmd
}

// usageSyncOutPath is where the snapshot goes: --out, else the configured
// synced_usage_path, else c3x-usage.synced.yml. It must never be the
// hand-written usage file, which sync promises not to touch.
func usageSyncOutPath(out string, resolved config.Resolved) (string, error) {
	p := out
	if p == "" {
		p = resolved.SyncedUsagePath
	}
	if p == "" {
		p = config.DefaultSyncedUsageFile
	}
	if resolved.UsagePath != "" && sameFile(p, resolved.UsagePath) {
		return "", fmt.Errorf("%s is the hand-written usage file (usage_path); usage sync only writes the generated one", p)
	}
	return p, nil
}

func sameFile(a, b string) bool {
	if ai, err := os.Stat(a); err == nil {
		if bi, err := os.Stat(b); err == nil {
			return os.SameFile(ai, bi)
		}
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	return errA == nil && errB == nil && absA == absB
}

func supportedKinds(s usagesync.Source) []string {
	var kinds []string
	for k := range s.Emits() {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return kinds
}

func sortedErrorKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
