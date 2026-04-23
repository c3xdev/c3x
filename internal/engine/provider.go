package engine

import "github.com/c3xdev/c3x/internal/settings"

type Vendor interface {
	Type() string
	DisplayType() string
	ProjectName() string
	RelativePath() string
	VarFiles() []string
	AddMetadata(*WorkspaceMeta)
	LoadResources(ConsumptionMap) ([]*Workspace, error)
	Context() *settings.ProjectSession
}
