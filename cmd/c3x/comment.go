package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/open-policy-agent/opa/ast"  //nolint:staticcheck // we need to use this deprecated package to support parsing of rego policies
	"github.com/open-policy-agent/opa/rego" //nolint:staticcheck // we need to use this deprecated package to support parsing of rego policies
	"github.com/spf13/cobra"

	"github.com/c3xdev/c3x/internal/apiclient"
	"github.com/c3xdev/c3x/internal/logging"

	"github.com/c3xdev/c3x/internal/clierror"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/c3xdev/c3x/internal/settings"
)

type CommentOutput struct {
	Body           string
	HasDiff        bool
	ValidAt        *time.Time
	AddRunResponse apiclient.AddRunResponse
}

var (
	validCommentOutputFormats = []string{
		"json",
	}
)

func commentCmd(ctx *settings.Session) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Post a C3X comment to GitHub, GitLab, Azure Repos or Bitbucket",
		Long:  "Post a C3X comment to GitHub, GitLab, Azure Repos or Bitbucket",
		Example: `  Update the C3X comment on a GitHub pull request:

      c3x comment github --repo my-org/my-repo --pull-request 3 --path c3x.json --behavior update --github-token $GITHUB_TOKEN

  Delete old C3X comments and post a new comment to a GitLab commit:

      c3x comment gitlab --repo my-org/my-repo --commit 2ca7182 --path c3x.json --behavior delete-and-new --gitlab-token $GITLAB_TOKEN

  Post a new comment to an Azure Repos pull request:

      c3x comment azure-repos --repo-url https://dev.azure.com/my-org/my-project/_git/my-repo --pull-request 3 --path c3x.json --behavior new --azure-access-token $AZURE_ACCESS_TOKEN`,
		ValidArgs: []string{"--", "-"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmds := []*cobra.Command{commentGitHubCmd(ctx), commentGitLabCmd(ctx), commentAzureReposCmd(ctx), commentBitbucketCmd(ctx)}
	for _, subCmd := range cmds {
		subCmd.RunE = checkAPIKeyIsValid(ctx, subCmd.RunE)

		subCmd.Flags().StringArray("policy-path", nil, "Path to C3X policy files, glob patterns need quotes (experimental)")
		subCmd.Flags().Bool("show-all-projects", false, "Show all projects in the table of the comment output")
		subCmd.Flags().Bool("show-changed", false, "Show only projects in the table that have code changes")
		subCmd.Flags().Bool("show-skipped", true, "List unsupported resources")
		_ = subCmd.Flags().MarkHidden("show-changed")
		subCmd.Flags().Bool("skip-no-diff", false, "Skip posting comment if there are no resource changes. Only applies to update, hide-and-new, and delete-and-new behaviors")
		_ = subCmd.Flags().MarkHidden("skip-no-diff")
		subCmd.Flags().String("comment-path", "", "Path to comment content file (experimental)")
		_ = subCmd.Flags().MarkHidden("comment-path")
	}

	cmd.AddCommand(cmds...)

	return cmd
}

func buildCommentOutput(cmd *cobra.Command, ctx *settings.Session, paths []string, mdOpts render.MarkdownOptions) (*CommentOutput, error) {
	inputs, err := render.LoadPaths(paths)
	if err != nil {
		return nil, err
	}

	combined, err := render.Combine(inputs)
	if errors.As(err, &clierror.WarningError{}) {
		logging.Logger.Warn().Msg(err.Error())
	} else if err != nil {
		return nil, err
	}

	combined.IsCIRun = ctx.IsCIRun()

	var commentData string
	var governanceFailures render.GovernanceFailures
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	var result apiclient.AddRunResponse
	if ctx.IsCloudUploadEnabled() && !dryRun {
		if ctx.Config.IsSelfHosted() {
			logging.Logger.Warn().Msg("C3X Cloud is part of C3X's hosted services. Contact hello@c3x.dev for help.")
		} else {
			combined.Metadata.C3XCommand = "comment"
			result = shareCombinedRun(ctx, combined, inputs)
			combined.RunID, combined.ShareURL, combined.CloudURL, governanceFailures = result.RunID, result.ShareURL, result.CloudURL, result.GovernanceFailures
			commentData = result.CommentMarkdown
		}
	}

	var out *CommentOutput

	commentPath, _ := cmd.Flags().GetString("comment-path")
	if commentPath != "" {
		commentData, err = render.LoadCommentData(commentPath)
		if err != nil {
			return nil, fmt.Errorf("Error loading %s used by --comment-path flag. %s", commentPath, err)
		}
	}

	if commentData != "" {
		// the full comment markdown has been received from the API addRun or loaded from the comment-path file,
		// so use that instead of building the output using the render.ToMarkdown templates.
		out = &CommentOutput{
			Body:           commentData,
			HasDiff:        combined.HasDiff(),
			ValidAt:        &combined.TimeGenerated,
			AddRunResponse: result,
		}
	}

	var policyChecks render.PolicyCheck
	policyPaths, _ := cmd.Flags().GetStringArray("policy-path")
	if len(policyPaths) > 0 {
		policyChecks, err = queryPolicy(policyPaths, combined)
		if err != nil {
			return nil, err
		}

		ctx.ContextValues.SetValue("passedPolicyCount", len(policyChecks.Passed))
		ctx.ContextValues.SetValue("failedPolicyCount", len(policyChecks.Failures))
	}

	if out == nil {
		opts := render.Options{
			DashboardEndpoint: ctx.Config.DashboardEndpoint,
			NoColor:           ctx.Config.NoColor,
			PolicyOutput:      render.NewPolicyOutput(policyChecks),
		}
		opts.ShowAllProjects, _ = cmd.Flags().GetBool("show-all-projects")
		opts.ShowOnlyChanges, _ = cmd.Flags().GetBool("show-changed")
		opts.ShowSkipped, _ = cmd.Flags().GetBool("show-skipped")

		md, err := render.ToMarkdown(combined, opts, mdOpts)
		if err != nil {
			return nil, err
		}

		b := md.Msg
		ctx.ContextValues.SetValue("truncated", md.OriginalMsgSize != md.RuneLen)
		ctx.ContextValues.SetValue("originalLength", md.OriginalMsgSize)

		out = &CommentOutput{
			Body:           string(b),
			HasDiff:        combined.HasDiff(),
			ValidAt:        &combined.TimeGenerated,
			AddRunResponse: result,
		}
	}

	if policyChecks.HasFailed() {
		return out, policyChecks.Failures
	}
	if len(governanceFailures) > 0 {
		return out, governanceFailures
	}

	return out, nil
}

type PRNumber int

func (p *PRNumber) Set(value string) error {
	if value == "" {
		return nil
	}

	v, err := strconv.Atoi(value)
	*p = PRNumber(v)

	if err != nil {
		return errors.New("must be integer")
	}

	return nil
}

func (p *PRNumber) String() string {
	return fmt.Sprintf("%d", *p)
}

func (p *PRNumber) Type() string {
	return "int"
}

func queryPolicy(policyPaths []string, input render.Report) (render.PolicyCheck, error) {
	checks := render.PolicyCheck{
		Enabled: true,
	}

	inputValue, err := ast.InterfaceToValue(input)
	if err != nil {
		return checks, fmt.Errorf("Unable to process C3X output into Rego input: %s", err.Error())
	}

	ctx := context.Background()
	r := rego.New(
		rego.Query("data.c3x.deny"),
		rego.ParsedInput(inputValue),
		rego.Load(policyPaths, func(abspath string, info os.FileInfo, depth int) bool {
			return false
		}),
	)
	pq, err := r.PrepareForEval(ctx)
	if err != nil {
		return checks, fmt.Errorf("Unable to query provided policies: %s", err.Error())
	}

	res, err := pq.Eval(ctx)
	if err != nil {
		return checks, err
	}

	if len(res) == 0 {
		return checks, fmt.Errorf("The provided polices returned no valid data.c3x.deny rules. Please check that the policies are formatted correctly.")
	}

	for _, e := range res[0].Expressions {
		switch v := e.Value.(type) {
		case map[string]interface{}:
			readPolicyOut(v, &checks)
		case []interface{}:
			for _, ii := range v {
				if m, ok := ii.(map[string]interface{}); ok {
					readPolicyOut(m, &checks)
				}
			}
		}
	}

	return checks, nil
}

func readPolicyOut(v map[string]interface{}, checks *render.PolicyCheck) {
	msg, ok := v["msg"].(string)
	if !ok {
		checks.Failures = append(checks.Failures, "Policy rule invalid as it did not contain {msg: string} property in output object. Please edit rule output object.")
		return
	}

	failed, ok := v["failed"].(bool)
	if !ok {
		checks.Failures = append(checks.Failures, fmt.Sprintf("Policy rule: [%s] has {failed} but it is not a bool. Please edit rule output object.", msg))
		return
	}

	if failed {
		checks.Failures = append(checks.Failures, msg)
		return
	}

	checks.Passed = append(checks.Passed, msg)
}

func isErrorUnhandled(err error) bool {
	if err == nil {
		return false
	}

	switch err.(type) {
	case render.PolicyCheckFailures, render.GovernanceFailures:
		return false
	}

	return true
}

// loadTLSConfigFromEnv creates a TLS config with CA certificates loaded from
// GIT_SSL_CAINFO environment variable if set.
func loadTLSConfigFromEnv(ctx *settings.Session) (*tls.Config, error) {
	tlsConfig := &tls.Config{} // nolint: gosec

	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		logging.Logger.Warn().Err(err).Msg("could not load system cert pool, using empty pool")
	}
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Load CA certificates from GIT_SSL_CAINFO if set
	if ctx.Config.TLSCACertFile != "" {
		caCerts, err := os.ReadFile(ctx.Config.TLSCACertFile)
		if err != nil {
			return nil, fmt.Errorf("error reading CA cert file %s: %w", ctx.Config.TLSCACertFile, err)
		}
		ok := rootCAs.AppendCertsFromPEM(caCerts)
		if !ok {
			logging.Logger.Warn().Msg("No CA certs appended, only using system certs")
		} else {
			logging.Logger.Debug().Msgf("Loaded CA certs from %s", ctx.Config.TLSCACertFile)
		}
	}

	tlsConfig.RootCAs = rootCAs
	return tlsConfig, nil
}
