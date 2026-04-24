package main

import (
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/spf13/cobra"
)

func configCmd(ctx *settings.Session) *cobra.Command {
	cmd := configureCmd(ctx)
	cmd.Use = "config"
	cmd.Short = "Display or change global configuration"
	cmd.Aliases = []string{}
	return cmd
}
