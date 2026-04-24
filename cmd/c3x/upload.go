package main

import (
	"fmt"
	"strings"

	jsoniter "github.com/json-iterator/go"

	"github.com/spf13/cobra"

	"github.com/c3xdev/c3x/internal/apiclient"
	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/c3xdev/c3x/internal/ui"
)

var (
	validUploadOutputFormats = []string{
		"json",
	}
)

func uploadCmd(ctx *settings.Session) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a C3X JSON file to C3X Cloud",
		Long: `Upload a C3X JSON file to C3X Cloud. This is useful if you
do not use 'c3x comment' and instead want to define run metadata,
such as pull request URL or title, and upload the results manually.

See https://c3x.dev/docs/features/cli_commands/#upload-runs`,
		Example: `  Upload a C3X JSON file:
      export C3X_VCS_PULL_REQUEST_URL=http://github.com/myorg...
      export C3X_VCS_PULL_REQUEST_TITLE="My PR title"
      # ... other env vars here

      c3x diff --path plan.json --format json --out-file c3x.json

      c3x upload --path c3x.json`,
		ValidArgs: []string{"--", "-"},
		RunE: checkAPIKeyIsValid(ctx, func(cmd *cobra.Command, args []string) error {
			var err error

			format, _ := cmd.Flags().GetString("format")
			format = strings.ToLower(format)
			if format != "" && !contains(validUploadOutputFormats, format) {
				ui.PrintUsage(cmd)
				return fmt.Errorf("--format only supports %s", strings.Join(validUploadOutputFormats, ", "))
			}

			if ctx.Config.IsSelfHosted() {
				return fmt.Errorf("C3X Cloud is part of C3X's hosted services. Contact hello@c3x.dev for help.")
			}

			path, _ := cmd.Flags().GetString("path")

			root, err := render.Load(path)
			if err != nil {
				return fmt.Errorf("could not load input file %s err: %w", path, err)
			}

			dashboardClient := apiclient.NewDashboardAPIClient(ctx)
			result, err := dashboardClient.AddRun(ctx, root)
			if err != nil {
				return fmt.Errorf("failed to upload to C3X Cloud: %w", err)
			}

			if format == "json" {
				b, err := jsoniter.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal result: %w", err)
				}
				cmd.Print(string(b))
			} else if result.ShareURL != "" {
				cmd.Println("Share this cost estimate: ", ui.LinkString(result.ShareURL))
			}

			pricingClient := apiclient.GetPricingAPIClient(ctx)
			err = pricingClient.AddEvent("c3x-upload", ctx.EventEnv())
			if err != nil {
				logging.Logger.Debug().Err(err).Msg("could not report event")
			}

			return nil
		}),
	}

	cmd.Flags().StringP("path", "p", "", "Path to C3X JSON file.")
	cmd.Flags().String("format", "", "Output format: json")

	_ = cmd.MarkFlagRequired("path")
	_ = cmd.MarkFlagFilename("path", "json")
	return cmd
}
