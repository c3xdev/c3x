package terraform

import (
	"os"

	"github.com/pkg/errors"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/settings"
)

type StateJSONProvider struct {
	ctx                  *settings.ProjectSession
	Path                 string
	includePastResources bool
}

func NewStateJSONProvider(ctx *settings.ProjectSession, includePastResources bool) engine.Vendor {
	return &StateJSONProvider{
		ctx:                  ctx,
		Path:                 ctx.ProjectConfig.Path,
		includePastResources: includePastResources,
	}
}

func (p *StateJSONProvider) ProjectName() string {
	return settings.CleanProjectName(p.ctx.ProjectConfig.Path)
}

func (p *StateJSONProvider) VarFiles() []string {
	return nil
}

func (p *StateJSONProvider) RelativePath() string {
	return p.ctx.ProjectConfig.Path
}

func (p *StateJSONProvider) Context() *settings.ProjectSession { return p.ctx }

func (p *StateJSONProvider) Type() string {
	return "terraform_state_json"
}

func (p *StateJSONProvider) DisplayType() string {
	return "Terraform state JSON file"
}

func (p *StateJSONProvider) AddMetadata(metadata *engine.WorkspaceMeta) {
	metadata.ConfigSha = p.ctx.ProjectConfig.ConfigSha
}

func (p *StateJSONProvider) LoadResources(usage engine.ConsumptionMap) ([]*engine.Workspace, error) {
	logging.Logger.Debug().Msg("Extracting only cost-related params from terraform")

	j, err := os.ReadFile(p.Path)
	if err != nil {
		return []*engine.Workspace{}, errors.Wrap(err, "Error reading Terraform state JSON file")
	}

	metadata := engine.DetectProjectMetadata(p.ctx.ProjectConfig.Path)
	metadata.Type = p.Type()
	p.AddMetadata(metadata)
	name := p.ctx.ProjectConfig.Name
	if name == "" {
		name = metadata.GenerateProjectName(p.ctx.RunContext.VCSMetadata.Remote, p.ctx.RunContext.IsCloudEnabled())
	}

	project := engine.NewProject(name, metadata)
	parser := NewParser(p.ctx, p.includePastResources)

	j, _ = StripSetupTerraformWrapper(j)
	parsedConf, err := parser.parseJSON(j, usage)
	if err != nil {
		return []*engine.Workspace{project}, errors.Wrap(err, "Error parsing Terraform state JSON file")
	}

	project.AddProviderMetadata(parsedConf.ProviderMetadata)

	project.PartialPastResources = parsedConf.PastResources
	project.PartialResources = parsedConf.CurrentResources

	return []*engine.Workspace{project}, nil
}
