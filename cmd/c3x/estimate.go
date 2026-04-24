package main

import (
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/spf13/cobra"
)

func estimateCmd(ctx *settings.Session) *cobra.Command {
	cmd := breakdownCmd(ctx)
	cmd.Use = "estimate"
	cmd.Short = "Estimate cloud infrastructure costs"
	cmd.Long = "Estimate cloud infrastructure costs from Terraform, Terragrunt, or CloudFormation"
	cmd.Example = `  Use Terraform directory:

      c3x estimate --path /code --terraform-var-file my.tfvars

  Use Terraform plan JSON:

      terraform plan -out tfplan.binary
      terraform show -json tfplan.binary > plan.json
      c3x estimate --path plan.json`
	cmd.Aliases = []string{}
	return cmd
}
