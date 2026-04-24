package settings

import (
	"context"
	"sync"

	"github.com/rs/zerolog"

	"github.com/c3xdev/c3x/internal/logging"
)

type ProjectSessioner interface {
	ProjectContext() map[string]interface{}
}

type ProjectSession struct {
	RunContext    *Session
	ProjectConfig *Project
	logger        zerolog.Logger
	ContextValues *ContextValues
	mu            *sync.RWMutex

	UsingCache bool
	CacheErr   string
}

func NewProjectContext(runCtx *Session, projectCfg *Project, logFields interface{}) *ProjectSession {
	ctx := logging.Logger.With().
		Str("project_name", projectCfg.Name).
		Str("project_path", projectCfg.Path)

	if logFields != nil {
		switch v := logFields.(type) {
		case context.Context:
			ctx = ctx.Ctx(v)
		default:
			ctx = ctx.Fields(v)
		}
	}

	contextLogger := ctx.Logger()

	return &ProjectSession{
		RunContext:    runCtx,
		ProjectConfig: projectCfg,
		logger:        contextLogger,
		ContextValues: NewContextValues(map[string]interface{}{}),
		mu:            &sync.RWMutex{},
	}
}

func (c *ProjectSession) SetProjectType(projectType string) {
	c.ContextValues.SetValue("project_type", projectType)
	var projectTypes []interface{}
	if t, ok := c.RunContext.ContextValues.GetValue("projectTypes"); ok {
		projectTypes = t.([]interface{})
	}

	projectTypes = append(projectTypes, projectType)
	c.RunContext.ContextValues.SetValue("projectTypes", projectTypes)
}

func (c *ProjectSession) Logger() zerolog.Logger {
	return c.logger
}

func (c *ProjectSession) SetFrom(d ProjectSessioner) {
	m := d.ProjectContext()
	for k, v := range m {
		c.ContextValues.SetValue(k, v)
	}
}
