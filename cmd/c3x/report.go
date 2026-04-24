package main

import (
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/spf13/cobra"
)

func reportCmd(ctx *settings.Session) *cobra.Command {
	cmd := outputCmd(ctx)
	cmd.Use = "report"
	cmd.Short = "Combine and format C3X JSON files in different formats"
	cmd.Aliases = []string{}
	return cmd
}
