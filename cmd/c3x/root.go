// Command c3x is the CLI entry point. The package is intentionally thin:
// each subcommand wires its dependencies, calls the relevant internal
// module, and renders the result. All real work lives in `internal/`.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/c3xdev/c3x/internal/observability"
	"github.com/spf13/cobra"
)

// version is set by goreleaser via -ldflags. Falls back to "dev" for
// local builds so `c3x --version` always reports something useful.
var version = "dev"

// newRootCmd assembles the cobra tree. Returned (not assigned to a
// package-level var) so tests can build a fresh command graph per case.
func newRootCmd() *cobra.Command {
	var verbosity int

	root := &cobra.Command{
		Use:           "c3x",
		Short:         "Cloud cost estimation for Terraform and CloudFormation.",
		Long:          longDescription,
		Version:       version,
		SilenceUsage:  true, // we print our own error context
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			observability.Configure(verbosity)
		},
	}
	root.PersistentFlags().CountVarP(&verbosity, "verbose", "v",
		"increase log verbosity (-v info, -vv debug, -vvv with source locations)")

	root.AddCommand(
		newEstimateCmd(),
		newDiffCmd(),
		newTopCmd(),
		newRecommendCmd(),
		newCommentCmd(),
		newPricingCmd(),
		newVersionCmd(),
		newSupportedResourcesCmd(),
		newDoctorCmd(),
		newConfigureCmd(),
		newPolicyCmd(),
	)
	return root
}

func main() {
	// Cancel the root context on Ctrl+C / SIGTERM so subcommands
	// reading from cmd.Context() abort cleanly instead of hanging
	// on a pending HTTP call.
	// os.Exit skips defers, so the signal handler is released
	// explicitly on every path rather than deferred.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	root := newRootCmd()
	err := root.ExecuteContext(ctx)
	stop()
	if err == nil {
		return
	}
	// Sentinel errors that already printed a user-facing message
	// (typically with helpful diff/total numbers) — exit non-zero
	// without re-printing.
	if errors.Is(err, errBudgetExceeded) || errors.Is(err, errBudgetDeltaExceeded) {
		os.Exit(1)
	}
	// Cobra has SilenceErrors=true so we print the chain ourselves.
	fmt.Fprintf(os.Stderr, "c3x: %v\n", err)
	os.Exit(1)
}

const longDescription = `c3x estimates the monthly cost of cloud infrastructure described in
Terraform or CloudFormation, before you apply it.

It pulls live prices from pricing.c3x.dev (no API key, no SaaS), caches
them locally, and renders a per-resource breakdown that you can paste
into a PR, gate on a budget, or feed into another tool.

  c3x estimate                    # show monthly cost of ./
  c3x diff --baseline base.json   # compare against a saved baseline
  c3x recommend                   # suggest cheaper alternatives
  c3x comment github              # post the breakdown as a PR comment`
