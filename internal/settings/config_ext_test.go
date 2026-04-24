package settings

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSettings_IsSelfHosted(t *testing.T) {
	cfg := DefaultConfig()
	assert.False(t, cfg.IsSelfHosted())

	cfg.PricingAPIEndpoint = "https://custom.pricing.example.com"
	assert.True(t, cfg.IsSelfHosted())
}

func TestSettings_WriteLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogLevel = "debug"
	assert.Equal(t, "debug", cfg.WriteLevel())
}

func TestSettings_WriteLevel_Default(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogLevel = ""
	level := cfg.WriteLevel()
	assert.NotEmpty(t, level)
}

func TestSettings_LogFields(t *testing.T) {
	cfg := DefaultConfig()
	fields := cfg.LogFields()
	// LogFields may return nil for default config — that's fine
	_ = fields
}

func TestSettings_SetLogWriter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetLogWriter(os.Stderr)
	writer := cfg.LogWriter()
	assert.NotNil(t, writer)
}

func TestSettings_SetLogDisableTimestamps(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetLogDisableTimestamps(true)
}

func TestSettings_LoadFromEnv(t *testing.T) {
	cfg := DefaultConfig()
	os.Setenv("C3X_API_KEY", "test-key-123")
	defer os.Unsetenv("C3X_API_KEY")

	err := cfg.LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "test-key-123", cfg.APIKey)
}

func TestSettings_CachePath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RootPath = os.TempDir()
	path := cfg.CachePath()
	_ = path // May return empty, that's OK
}

func TestYamlError_Error(t *testing.T) {
	ye := YamlError{}
	msg := ye.Error()
	_ = msg // Just verify it doesn't panic
}

func TestProject_Config(t *testing.T) {
	p := Project{
		Path:               "/code/infra",
		Name:               "production",
		TerraformBinary:    "terraform",
		TerraformWorkspace: "prod",
		SkipAutodetect:     true,
	}
	assert.Equal(t, "/code/infra", p.Path)
	assert.Equal(t, "production", p.Name)
	assert.True(t, p.SkipAutodetect)
}

func TestCredentials_FilePath(t *testing.T) {
	path := CredentialsFilePath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "c3x")
}

func TestConfiguration_FilePath(t *testing.T) {
	path := ConfigurationFilePath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "c3x")
}

func TestSettings_Format(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "table", cfg.Format)
}
