package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSettings_WorkingDirectory_WithConfigFile(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ConfigFilePath = "/code/c3x.yml"

	wd := cfg.WorkingDirectory()
	assert.NotEmpty(t, wd)
}

func TestSettings_WorkingDirectory_WithRootPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RootPath = "/code/infra"

	wd := cfg.WorkingDirectory()
	assert.Equal(t, "/code/infra", wd)
}

func TestUserConfigDir(t *testing.T) {
	dir := userConfigDir()
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "c3x")
}

func TestSettings_Fields(t *testing.T) {
	cfg := DefaultConfig()

	// Verify key default values
	assert.Equal(t, "table", cfg.Format)
	assert.NotEmpty(t, cfg.DefaultPricingAPIEndpoint)
	assert.Equal(t, ".c3x", C3XDir)
}

func TestSettings_Parallelism(t *testing.T) {
	cfg := DefaultConfig()
	// Default parallelism should be nil (auto-detect)
	assert.Nil(t, cfg.Parallelism)
}

func TestSettings_NoCache(t *testing.T) {
	cfg := DefaultConfig()
	assert.False(t, cfg.NoCache)
}

func TestSettings_ShowAllProjects(t *testing.T) {
	cfg := DefaultConfig()
	assert.False(t, cfg.ShowAllProjects)
}

func TestPathOverride_Fields(t *testing.T) {
	po := PathOverride{
		Path: "/override/path",
	}
	assert.Equal(t, "/override/path", po.Path)
}
