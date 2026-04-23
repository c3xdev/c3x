package cloudformation

import (
	"github.com/awslabs/goformation/v7"
	"github.com/pkg/errors"

	"github.com/c3xdev/c3x/internal/settings"
	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/engine"
)

type TemplateProvider struct {
	ctx                  *settings.ProjectSession
	Path                 string
	includePastResources bool
}

func NewTemplateProvider(ctx *settings.ProjectSession, includePastResources bool) engine.Vendor {
	return &TemplateProvider{
		ctx:                  ctx,
		Path:                 ctx.ProjectConfig.Path,
		includePastResources: includePastResources,
	}
}

func (p *TemplateProvider) ProjectName() string {
	return settings.CleanProjectName(p.ctx.ProjectConfig.Path)
}

func (p *TemplateProvider) VarFiles() []string {
	return nil
}

func (p *TemplateProvider) Context() *settings.ProjectSession { return p.ctx }

func (p *TemplateProvider) Type() string {
	return "cloudformation"
}

func (p *TemplateProvider) DisplayType() string {
	return "CloudFormation"
}

func (p *TemplateProvider) AddMetadata(metadata *engine.WorkspaceMeta) {
	metadata.ConfigSha = p.ctx.ProjectConfig.ConfigSha
}

func (p *TemplateProvider) RelativePath() string {
	return p.ctx.ProjectConfig.Path
}

func (p *TemplateProvider) LoadResources(usage engine.ConsumptionMap) ([]*engine.Workspace, error) {
	template, err := goformation.Open(p.Path)
	if err != nil {
		return []*engine.Workspace{}, errors.Wrap(err, "Error reading CloudFormation template file")
	}

	logging.Logger.Debug().Msg("Extracting only cost-related params from cloudformation")

	metadata := engine.DetectProjectMetadata(p.ctx.ProjectConfig.Path)
	metadata.Type = p.Type()
	p.AddMetadata(metadata)
	name := p.ctx.ProjectConfig.Name
	if name == "" {
		name = metadata.GenerateProjectName(p.ctx.RunContext.VCSMetadata.Remote, p.ctx.RunContext.IsCloudEnabled())
	}

	project := engine.NewProject(name, metadata)
	parser := NewParser(p.ctx, p.includePastResources)
	parsed := parser.parseTemplate(template, usage)
	if err != nil {
		return []*engine.Workspace{project}, errors.Wrap(err, "Error parsing CloudFormation template file")
	}

	for _, item := range parsed {
		project.PartialResources = append(project.PartialResources, item.PartialResource)
	}

	return []*engine.Workspace{project}, nil
}
