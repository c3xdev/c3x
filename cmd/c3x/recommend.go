package main

import (
	"encoding/json"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/recommend"
	"github.com/c3xdev/c3x/internal/render"
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/spf13/cobra"
)

func recommendCmd(ctx *settings.Session) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recommend",
		Short: "Suggest cost optimizations for estimated resources",
		Long:  "Analyze estimated cloud resources and suggest cost optimizations such as newer instance generations, better storage types, and architectural improvements.",
		Example: `  Show recommendations for a Terraform project:

      c3x recommend --path /code

  Show recommendations in JSON format:

      c3x recommend --path /code --format json

  Run estimate with inline recommendations:

      c3x estimate --path /code --recommend`,
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
				return err
			}

			return runRecommend(cmd, ctx)
		}),
	}

	addRunFlags(cmd)
	newEnumFlag(cmd, "format", "table", "Output format", []string{"json", "table"})

	return cmd
}

func runRecommend(cmd *cobra.Command, ctx *settings.Session) error {
	pr, err := newParallelRunner(cmd, ctx)
	if err != nil {
		return err
	}

	projectResults, err := pr.run()
	if err != nil {
		return err
	}

	pr.pricingFetcher.LogWarnings()

	projects := projectsFromResults(projectResults)

	// Run cost calculations first
	r, err := render.ToOutputFormat(ctx.Config, projects)
	if err != nil {
		return err
	}
	_ = r

	// Analyze for recommendations
	result := recommend.Analyze(projects)

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		cmd.Println(string(b))
	default:
		cmd.PrintErrln()
		cmd.Print(recommend.FormatTable(result, ctx.Config.NoColor))
	}

	return nil
}

func projectsFromResults(results []projectResult) []*engine.Workspace {
	projects := make([]*engine.Workspace, 0)
	for _, r := range results {
		projects = append(projects, r.projectOut.projects...)
	}
	return projects
}
