package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	stdLog "log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/pkg/errors"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/c3xdev/c3x/internal/apiclient"
	"github.com/c3xdev/c3x/internal/clierror"
	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/c3xdev/c3x/internal/ui"
	"github.com/c3xdev/c3x/internal/update"
	"github.com/c3xdev/c3x/internal/version"
)

func init() {
	// set the stdlib default logger to flush to discard, this is done as a number of
	// Terraform libs use the std logger directly, which impacts C3X output.
	stdLog.SetOutput(io.Discard)
}

func main() {
	if os.Getenv("C3X_MEMORY_PROFILE") == "true" {
		defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()
	} else if os.Getenv("C3X_CPU_PROFILE") == "true" {
		defer profile.Start(profile.CPUProfile, profile.ProfilePath(".")).Stop()
	}

	Run(nil, nil)
	err := apiclient.GetPricingAPIClient(nil).FlushCache()
	if err != nil {
		logging.Logger.Debug().Err(err).Msg("could not flush pricing API cache to filesystem")
	}
}

// Run starts the C3X application with the configured cobra cmds.
// Cmd args and flags are parsed from the cli, but can also be directly injected
// using the modifyCtx and args parameters.
func Run(modifyCtx func(*settings.Session), args *[]string) {
	ctx, err := settings.NewRunContextFromEnv(context.Background())
	if err != nil {
		if err.Error() != "" {
			ui.PrintError(ctx.ErrWriter, err.Error())
		}

		ctx.Exit(1)
	}

	if modifyCtx != nil {
		modifyCtx(ctx)
	}

	var appErr error
	updateMessageChan := make(chan *update.Info)

	defer func() {
		if appErr != nil {
			if v, ok := appErr.(*clierror.PanicError); ok {
				handleUnexpectedErr(ctx, v)
			} else {
				handleCLIError(ctx, appErr)
			}
		}

		unexpectedErr := recover()
		if unexpectedErr != nil {
			panicErr := clierror.NewPanicError(fmt.Errorf("%s", unexpectedErr), debug.Stack())
			handleUnexpectedErr(ctx, panicErr)
		}

		handleUpdateMessage(updateMessageChan)

		if appErr != nil || unexpectedErr != nil {
			ctx.Exit(1)
		}
	}()

	startUpdateCheck(ctx, updateMessageChan)

	rootCmd := newRootCmd(ctx)
	if args != nil {
		rootCmd.SetArgs(*args)
	}

	appErr = rootCmd.Execute()
}

type debugWriter struct {
	f *os.File
}

func (d debugWriter) Write(p []byte) (n int, err error) {
	p = bytes.Trim(p, " \n\t")
	return d.f.Write(append(p, []byte("\n")...))
}

func newRootCmd(ctx *settings.Session) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "c3x",
		Version: version.Version,
		Short:   "Cloud cost estimates for Terraform",
		Long: fmt.Sprintf(`C3X - Cloud cost estimates for Terraform

%s
  Quick start: https://c3x.dev/docs
  Add cost estimates to your pull requests: https://c3x.dev/cicd`, ui.BoldString("DOCS")),
		Example: `  Estimate costs from Terraform directory:

      c3x estimate --path /code --terraform-var-file my.tfvars

  Show cost diff:

      c3x estimate --path /code --format json --out-file c3x-base.json
      # Make Terraform code changes
      c3x diff --path /code --compare-to c3x-base.json`,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			ctx.ContextValues.SetValue("command", cmd.Name())
			ctx.CMD = cmd.Name()
			if cmd.Name() == "comment" || (cmd.Parent() != nil && cmd.Parent().Name() == "comment") {
				ctx.SetIsC3XComment()
			}
			out, _ := cmd.Flags().GetBool("debug-report")
			if out {
				debugFile := "c3x-debug-report.json"
				var f *os.File
				var err error

				// 0600: owner-only read/write. Debug reports contain API bodies,
				// project names, and VCS metadata — do not make world-readable.
				f, err = os.OpenFile(debugFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)

				if err != nil {
					return fmt.Errorf("could not generate debug report file %w", err)
				}
				_, _ = f.WriteString("[\n")

				writer := debugWriter{f: f}
				ctx.ErrWriter = writer
				ctx.Config.SetLogWriter(writer)
			}
			err := loadGlobalFlags(ctx, cmd)
			if err != nil {
				return err
			}

			loadCloudSettings(ctx)

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Show the help
			return cmd.Help()
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			out, _ := cmd.Flags().GetBool("debug-report")
			if out {
				if f, ok := ctx.Config.LogWriter().(debugWriter); ok {
					_, _ = f.f.WriteString("{\"msg\":\"program finished\"}\n")

					_, _ = f.f.WriteString("]")
					_ = f.f.Close()
				}
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().Bool("no-color", false, "Turn off colored output")
	rootCmd.PersistentFlags().String("log-level", "", "Log level (trace, debug, info, warn, error, fatal)")
	rootCmd.PersistentFlags().Bool("debug-report", false, "Generate a debug report file which can be sent to C3X team")

	// Primary commands
	rootCmd.AddCommand(estimateCmd(ctx))
	rootCmd.AddCommand(diffCmd(ctx))
	rootCmd.AddCommand(recommendCmd(ctx))
	rootCmd.AddCommand(pricingCmd(ctx))
	rootCmd.AddCommand(reportCmd(ctx))
	rootCmd.AddCommand(commentCmd(ctx))
	rootCmd.AddCommand(uploadCmd(ctx))
	rootCmd.AddCommand(authCmd(ctx))
	rootCmd.AddCommand(configCmd(ctx))
	rootCmd.AddCommand(newGenerateCommand())
	rootCmd.AddCommand(completionCmd())

	// Hidden backward-compatible aliases
	breakdownAlias := breakdownCmd(ctx)
	breakdownAlias.Hidden = true
	rootCmd.AddCommand(breakdownAlias)

	outputAlias := outputCmd(ctx)
	outputAlias.Hidden = true
	rootCmd.AddCommand(outputAlias)

	configureAlias := configureCmd(ctx)
	configureAlias.Hidden = true
	rootCmd.AddCommand(configureAlias)

	registerAlias := registerCmd(ctx)
	registerAlias.Hidden = true
	rootCmd.AddCommand(registerAlias)

	rootCmd.AddCommand(figAutocompleteCmd())

	rootCmd.SetUsageTemplate(fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

%s{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`,
		ui.BoldString("USAGE"),
		ui.BoldString("ALIAS"),
		ui.BoldString("EXAMPLES"),
		ui.BoldString("AVAILABLE COMMANDS"),
		ui.BoldString("FLAGS"),
		ui.BoldString("GLOBAL FLAGS"),
		ui.BoldString("ADDITIONAL HELP TOPICS"),
	))

	rootCmd.SetVersionTemplate("C3X {{.Version}}\n")
	rootCmd.SetOut(ctx.OutWriter)
	rootCmd.SetErr(ctx.ErrWriter)

	return rootCmd
}

func startUpdateCheck(ctx *settings.Session, c chan *update.Info) {
	go func() {
		updateInfo, err := update.CheckForUpdate(ctx)
		if err != nil {
			logging.Logger.Debug().Err(err).Msg("error checking for C3X CLI update")
		}
		c <- updateInfo
		close(c)
	}()
}

func loadCloudSettings(ctx *settings.Session) {
	if ctx.Config.IsSelfHosted() || (ctx.Config.EnableCloud != nil && !*ctx.Config.EnableCloud) {
		return
	}

	dashboardClient := apiclient.NewDashboardAPIClient(ctx)
	result, err := dashboardClient.QueryCLISettings()
	if err != nil {
		logging.Logger.Debug().Err(err).Msg("Failed to load settings from C3X Cloud ")
		// ignore the error so the command can continue without failing
		return
	}
	logging.Logger.Debug().Str("result", fmt.Sprintf("%+v", result)).Msg("Successfully loaded settings from C3X Cloud")

	ctx.Config.EnableCloudForOrganization = result.CloudEnabled
	if result.UsageAPIEnabled && ctx.Config.UsageAPIEndpoint == "" {
		ctx.Config.UsageAPIEndpoint = ctx.Config.DashboardAPIEndpoint
		logging.Logger.Debug().Msg("Enabled usage API")
	}
	if result.ActualCostsEnabled && ctx.Config.UsageAPIEndpoint != "" {
		ctx.Config.UsageActualCosts = true
		logging.Logger.Debug().Msg("Enabled actual costs")
	}

	if (result.PoliciesAPIEnabled || result.TagsAPIEnabled) && ctx.Config.PolicyV2APIEndpoint == "" {
		ctx.Config.PolicyV2APIEndpoint = ctx.Config.DashboardAPIEndpoint
		logging.Logger.Debug().Msg("Using default policies V2 endpoint")
	}

	if result.PoliciesAPIEnabled {
		ctx.Config.PoliciesEnabled = true
		logging.Logger.Debug().Msg("Enabled policies V2")
	}

	if result.TagsAPIEnabled {
		ctx.Config.TagPoliciesEnabled = true
		logging.Logger.Debug().Msg("Enabled tag policies")
	}
}

func checkAPIKey(apiKey string, apiEndpoint string, defaultEndpoint string) error {
	// The default C3X pricing endpoint does not require an API key.
	return nil
}

var ignoredErrors = []string{
	"Policy check failed",
	"Governance check failed",
}

func handleCLIError(ctx *settings.Session, cliErr error) {
	if cliErr.Error() != "" {
		ui.PrintError(ctx.ErrWriter, cliErr.Error())
	}

	for _, pattern := range ignoredErrors {
		if strings.Contains(cliErr.Error(), pattern) {
			return
		}
	}

	err := apiclient.ReportCLIError(ctx, cliErr, true)
	if err != nil {
		logging.Logger.Debug().Err(err).Msg("error reporting CLI error")
	}
}

func handleUnexpectedErr(ctx *settings.Session, err error) {
	ui.PrintUnexpectedErrorStack(err)

	err = apiclient.ReportCLIError(ctx, err, false)
	if err != nil {
		logging.Logger.Debug().Err(err).Msg("error sending unexpected runtime error")
	}
}

func handleUpdateMessage(updateMessageChan chan *update.Info) {
	updateInfo := <-updateMessageChan
	if updateInfo != nil {
		msg := fmt.Sprintf("\n%s %s %s → %s\n%s\n",
			ui.WarningString("Update:"),
			"A new version of C3X is available:",
			ui.PrimaryString(version.Version),
			ui.PrimaryString(updateInfo.LatestVersion),
			ui.Indent(updateInfo.Cmd, "  "),
		)
		fmt.Fprint(os.Stderr, msg)
	}
}

func loadGlobalFlags(ctx *settings.Session, cmd *cobra.Command) error {
	if ctx.IsCIRun() {
		ctx.Config.NoColor = true
	}

	err := ctx.Config.LoadGlobalFlags(cmd)
	if err != nil {
		return err
	}

	ctx.ContextValues.SetValue("dashboardEnabled", ctx.Config.EnableDashboard)
	ctx.ContextValues.SetValue("cloudEnabled", ctx.IsCloudEnabled())
	ctx.ContextValues.SetValue("isDefaultPricingAPIEndpoint", ctx.Config.PricingAPIEndpoint == ctx.Config.DefaultPricingAPIEndpoint)

	flagNames := make([]string, 0)

	cmd.Flags().Visit(func(f *pflag.Flag) {
		flagNames = append(flagNames, f.Name)
	})

	ctx.ContextValues.SetValue("flags", flagNames)

	return nil
}

// saveOutFile saves the output of the command to the file path past in the `--out-file` flag
func saveOutFile(ctx *settings.Session, cmd *cobra.Command, outFile string, b []byte) error {
	return saveOutFileWithMsg(ctx, cmd, outFile, fmt.Sprintf("Output saved to %s", outFile), b)
}

// saveOutFile saves the output of the command to the file path past in the `--out-file` flag
func saveOutFileWithMsg(ctx *settings.Session, cmd *cobra.Command, outFile, successMsg string, b []byte) error {
	err := os.WriteFile(outFile, b, 0644) // nolint:gosec
	if err != nil {
		return errors.Wrap(err, "Unable to save output")
	}

	logging.Logger.Info().Msg(successMsg)

	return nil
}
