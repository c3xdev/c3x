package main

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/c3xdev/c3x/internal/apiclient"
	"github.com/c3xdev/c3x/internal/cloud"
	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/c3xdev/c3x/internal/ui"
)

func diffCmd(ctx *settings.Session) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff of monthly costs between current and planned state",
		Long:  "Show diff of monthly costs between current and planned state",
		Example: `  Use Terraform directory:

      c3x breakdown --path /code --format json --out-file c3x-base.json
      # Make Terraform code changes
      c3x diff --path /code --compare-to c3x-base.json

  Use Terraform plan JSON:

      terraform plan -out tfplan.binary
      terraform show -json tfplan.binary > plan.json
      c3x diff --path plan.json`,
		ValidArgs: []string{"--", "-"},
		RunE: checkAPIKeyIsValid(ctx, func(cmd *cobra.Command, args []string) error {
			if err := checkAPIKey(ctx.Config.APIKey, ctx.Config.PricingAPIEndpoint, ctx.Config.DefaultPricingAPIEndpoint); err != nil {
				return err
			}

			err := loadRunFlags(ctx.Config, cmd)
			if err != nil {
				return err
			}

			err = checkRunConfig(cmd.ErrOrStderr(), ctx.Config)
			if err != nil {
				ui.PrintUsage(cmd)
				return err
			}

			err = checkDiffConfig(ctx.Config)
			if err != nil {
				ui.PrintUsage(cmd)
				return err
			}

			return runDiff(cmd, ctx)
		}),
	}

	addRunFlags(cmd)

	cmd.Flags().String("compare-to", "", "Path to C3X JSON file to compare against")
	newEnumFlag(cmd, "format", "diff", "Output format", []string{"json", "diff"})
	cmd.Flags().String("out-file", "", "Save output to a file")
	cmd.Flags().Float64("budget", 0, "Monthly cost budget in configured currency. Exits with code 1 if exceeded")
	cmd.Flags().Float64("budget-increase", 0, "Maximum allowed cost increase percentage. Exits with code 1 if exceeded")

	return cmd
}

func runDiff(cmd *cobra.Command, ctx *settings.Session) error {
	if len(ctx.Config.Projects) > 0 {
		path := ctx.Config.Projects[0].Path

		// if the path provided is a C3X JSON we need to run a compare run
		current, err := render.Load(path)
		if err == nil {
			if ctx.Config.CompareTo == "" {
				return errors.New("Passing a C3X JSON as a --path argument is only valid using the --compare-to flag")
			}

			return runCompare(cmd, ctx, current)
		}
	}

	return runMain(cmd, ctx)
}

func runCompare(cmd *cobra.Command, ctx *settings.Session, current render.Report) error {
	prior, err := render.Load(ctx.Config.CompareTo)
	if err != nil {
		return fmt.Errorf("Error loading %s used by --compare-to flag. %s", ctx.Config.CompareTo, err)
	}

	combined, err := render.CompareTo(ctx.Config, current, prior)
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	b, err := render.FormatOutput(strings.ToLower(format), combined, render.Options{
		DashboardEndpoint: ctx.Config.DashboardEndpoint,
		ShowSkipped:       ctx.Config.ShowSkipped,
		NoColor:           ctx.Config.NoColor,
		Fields:            ctx.Config.Fields,
		CurrencyFormat:    ctx.Config.CurrencyFormat,
	})
	if err != nil {
		return err
	}

	pricingClient := apiclient.GetPricingAPIClient(ctx)
	err = pricingClient.AddEvent("c3x-run", ctx.EventEnv())
	if err != nil {
		logging.Logger.Debug().Err(err).Msg("could not report event")
	}

	if outFile, _ := cmd.Flags().GetString("out-file"); outFile != "" {
		return saveOutFile(ctx, cmd, outFile, b)
	}

	cmd.Println(string(b))

	if err := checkBudget(cmd, combined); err != nil {
		return err
	}

	return nil
}

func checkDiffConfig(cfg *settings.Settings) error {
	for _, projectConfig := range cfg.Projects {
		if projectConfig.TerraformUseState {
			return errors.New("terraform_use_state cannot be used with `c3x diff` as the Terraform state only contains the current state")
		}

		projectType := cloud.DetectProjectType(projectConfig.Path, projectConfig.TerraformForceCLI)
		if (projectType == cloud.ProjectTypeAutodetect) && cfg.CompareTo == "" {
			examplePath := "/code"
			if projectConfig.Path != "" {
				examplePath = projectConfig.Path
			}

			msg := fmt.Sprintf(`To show a diff:
  1. Generate a cost estimate baseline: %s
  2. Make a Terraform code change
  3. Generate a cost estimate diff: %s`,
				fmt.Sprintf("`c3x breakdown --path %s --format json --out-file c3x-base.json`", examplePath),
				fmt.Sprintf("`c3x diff --path %s --compare-to c3x-base.json`", examplePath),
			)
			return errors.New(msg)
		}
	}

	return nil
}
