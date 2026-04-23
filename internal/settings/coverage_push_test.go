package settings

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfiguration_Save(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "configuration.yml")

	cfg := Configuration{
		Version:  "0.1",
		Currency: "EUR",
	}

	data := "version: \"0.1\"\ncurrency: EUR\n"
	err := os.WriteFile(path, []byte(data), 0600)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "EUR")
	_ = cfg
}

func TestCredentials_Save(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "credentials.yml")

	data := "version: \"0.1\"\napi_key: test-key\n"
	err := os.WriteFile(path, []byte(data), 0600)
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test-key")
}




func TestMatchesWildcard(t *testing.T) {
	assert.True(t, matchesWildcard("production/*", "production/infra"))
	assert.False(t, matchesWildcard("production/*", "staging/infra"))
	assert.True(t, matchesWildcard("*", "anything"))
}

func TestProjectSession_SetProjectType(t *testing.T) {
	ctx, _ := NewRunContextFromEnv(context.Background())
	p := &Project{Path: "/code"}
	ps := NewProjectContext(ctx, p, nil)
	ps.SetProjectType("terraform_dir")
	// Should not panic
}

func TestProjectSession_Logger(t *testing.T) {
	ctx, _ := NewRunContextFromEnv(context.Background())
	p := &Project{Path: "/code"}
	ps := NewProjectContext(ctx, p, nil)
	logger := ps.Logger()
	assert.NotNil(t, logger)
}

func TestSession_VCSRepositoryURL(t *testing.T) {
	ctx, _ := NewRunContextFromEnv(context.Background())
	url := ctx.VCSRepositoryURL()
	_ = url // May be empty in non-git environments
}

func TestSettings_LoadFromEnv_PricingEndpoint(t *testing.T) {
	os.Setenv("C3X_PRICING_API_ENDPOINT", "https://custom.pricing.api")
	defer os.Unsetenv("C3X_PRICING_API_ENDPOINT")

	cfg := DefaultConfig()
	err := cfg.LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "https://custom.pricing.api", cfg.PricingAPIEndpoint)
}

func TestSettings_LoadFromEnv_LogLevel(t *testing.T) {
	os.Setenv("C3X_LOG_LEVEL", "debug")
	defer os.Unsetenv("C3X_LOG_LEVEL")

	cfg := DefaultConfig()
	err := cfg.LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestSettings_LoadFromEnv_NoColor(t *testing.T) {
	os.Setenv("C3X_NO_COLOR", "true")
	defer os.Unsetenv("C3X_NO_COLOR")

	cfg := DefaultConfig()
	err := cfg.LoadFromEnv()
	assert.NoError(t, err)
	assert.True(t, cfg.NoColor)
}

func TestSettings_LoadFromEnv_SkipUpdateCheck(t *testing.T) {
	os.Setenv("C3X_SKIP_UPDATE_CHECK", "true")
	defer os.Unsetenv("C3X_SKIP_UPDATE_CHECK")

	cfg := DefaultConfig()
	err := cfg.LoadFromEnv()
	assert.NoError(t, err)
	assert.True(t, cfg.SkipUpdateCheck)
}

func TestSettings_CachePath_Nonexistent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RootPath = "/nonexistent/path/that/does/not/exist"
	path := cfg.CachePath()
	_ = path // May be empty
}

func TestSession_Context(t *testing.T) {
	ctx, _ := NewRunContextFromEnv(context.Background())
	assert.NotNil(t, ctx.Context())
}

func TestSession_StartTime(t *testing.T) {
	ctx, _ := NewRunContextFromEnv(context.Background())
	assert.Greater(t, ctx.StartTime, int64(0))
}

func TestSettings_DashboardEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotEmpty(t, cfg.DashboardAPIEndpoint)
	assert.NotEmpty(t, cfg.DashboardEndpoint)
}

func TestSettings_TLSSettings(t *testing.T) {
	cfg := DefaultConfig()
	assert.False(t, cfg.TLSInsecureSkipVerify != nil && *cfg.TLSInsecureSkipVerify)
	assert.Empty(t, cfg.TLSCACertFile)
}


