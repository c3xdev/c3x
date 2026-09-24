package main

// `c3x policy eval` runs a Rego policy bundle against the current
// estimate. It uses the standard `deny`/`warn` Rego convention so
// common policy bundles port with minimal rewrite.

import (
	"fmt"
	"os"

	"github.com/c3xdev/c3x/internal/config"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/policy"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/spf13/cobra"
)

func newPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Run Rego policies against an estimate or diff.",
	}
	cmd.AddCommand(newPolicyEvalCmd())
	return cmd
}

func newPolicyEvalCmd() *cobra.Command {
	var (
		policyPath   string
		estimatePath string
		baselinePath string
		path         string
		varFiles     []string
		vars         []string
		offline      bool
	)
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Evaluate one or more Rego policies against the estimate.",
		Long: `Loads policies (file or directory of .rego files), evaluates
them against the estimate's JSON shape, and exits non-zero if any
deny[msg] rule produces a value. warn[msg] rules print to stderr
but do not fail.

Policy data model:

  input.estimate.project_total        number (currency-converted)
  input.estimate.currency             string
  input.estimate.resources[*].kind    string
  input.estimate.resources[*].name    string
  input.estimate.resources[*].monthly_cost  number
  input.diff.baseline_total           number   (when --baseline supplied)
  input.diff.current_total            number
  input.diff.total_delta              number
  input.diff.resources[*]             same shape with baseline/current/delta

Provide either --estimate <file.json> to evaluate a saved baseline
or --path <terraform> to compute a fresh estimate first.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			est, err := loadEstimateForPolicy(cmd, estimatePath, path, varFiles, vars, offline)
			if err != nil {
				return err
			}

			var diff *domain.Diff
			if baselinePath != "" {
				base, err := loadEstimateForPolicy(cmd, baselinePath, "", nil, nil, offline)
				if err != nil {
					return fmt.Errorf("loading baseline: %w", err)
				}
				d := domain.ComputeDiff(base, est)
				diff = &d
			}

			res, err := policy.Eval(cmd.Context(), policyPath, policy.BuildInput(est, diff))
			if err != nil {
				return err
			}
			for _, msg := range res.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warn: %s\n", msg)
			}
			for _, msg := range res.Denials {
				fmt.Fprintf(cmd.ErrOrStderr(), "deny: %s\n", msg)
			}
			if res.HasDenials() {
				return fmt.Errorf("policy denied with %d violation(s)", len(res.Denials))
			}
			fmt.Fprintln(cmd.OutOrStdout(), "policy passed")
			return nil
		},
	}
	cmd.Flags().StringVar(&policyPath, "policy", "",
		"path to a .rego file or directory of policies (required)")
	cmd.Flags().StringVar(&estimatePath, "estimate", "",
		"path to a saved estimate JSON (from `c3x estimate --save-baseline`)")
	cmd.Flags().StringVar(&baselinePath, "baseline", "",
		"path to a baseline estimate JSON; policy input includes input.diff")
	cmd.Flags().StringVar(&path, "path", "",
		"Terraform or OpenTofu input — compute a fresh estimate from this path")
	cmd.Flags().StringArrayVar(&varFiles, "var-file", nil, "tfvars file (repeatable)")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "variable override name=value (repeatable)")
	cmd.Flags().BoolVar(&offline, "offline", false, "use the offline pricing stub when computing a fresh estimate")
	_ = cmd.MarkFlagRequired("policy")
	return cmd
}

// loadEstimateForPolicy returns an Estimate from either a saved
// JSON file or a fresh c3x estimate of a Terraform path.
func loadEstimateForPolicy(cmd *cobra.Command, estimatePath, terraformPath string, varFiles, vars []string, offline bool) (domain.Estimate, error) {
	if estimatePath != "" {
		raw, err := os.ReadFile(estimatePath)
		if err != nil {
			return domain.Estimate{}, err
		}
		// Baselines are written by the render JSON contract
		// (`estimate --save-baseline`); decode through the same
		// contract — plain json.Unmarshal cannot round-trip the
		// Reference encoding.
		est, err := render.DecodeEstimate(raw)
		if err != nil {
			return domain.Estimate{}, fmt.Errorf("parsing %s: %w (run `c3x estimate --save-baseline %s` to produce this file)",
				estimatePath, err, estimatePath)
		}
		return est, nil
	}
	if terraformPath == "" {
		return domain.Estimate{}, fmt.Errorf("either --estimate or --path is required")
	}
	projectDir, err := resolveProjectDir(terraformPath)
	if err != nil {
		return domain.Estimate{}, err
	}
	flags := map[string]any{}
	if offline {
		flags["offline"] = true
	}
	resolved, err := config.Resolve(projectDir, flags)
	if err != nil {
		return domain.Estimate{}, fmt.Errorf("resolving config: %w", err)
	}
	return computeCurrent(cmd.Context(), terraformPath, resolved, varFiles, vars)
}
