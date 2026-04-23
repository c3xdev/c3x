package tf

import (
	"github.com/c3xdev/c3x/internal/hclstack/errtrace"
	"github.com/c3xdev/c3x/internal/hclstack/collections"
	"github.com/c3xdev/c3x/internal/hclstack/options"
	"github.com/c3xdev/c3x/internal/hclstack/pkg/cli"
)

const (
	CommandName = "terraform"
)

var (
	nativeTerraformCommands = []string{"apply", "console", "destroy", "env", "fmt", "get", "graph", "import", "init", "metadata", "output", "plan", "providers", "push", "refresh", "show", "taint", "test", "version", "validate", "untaint", "workspace", "force-unlock", "state"}
)

func NewCommand(opts *options.TerragruntOptions) *cli.Command {
	return &cli.Command{
		Name:     CommandName,
		HelpName: "*",
		Usage:    "Terragrunt forwards all other commands directly to Terraform",
		Action:   action(opts),
	}
}

func action(opts *options.TerragruntOptions) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		if opts.TerraformCommand == CommandNameDestroy {
			opts.CheckDependentModules = true
		}

		if !opts.DisableCommandValidation && !collections.ListContainsElement(nativeTerraformCommands, opts.TerraformCommand) {
			return errtrace.WithStackTrace(WrongTerraformCommand(opts.TerraformCommand))
		}

		return Run(opts.OptionsFromContext(ctx))
	}
}
