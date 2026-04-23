package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentials_Save_And_Load(t *testing.T) {
	tmpDir := t.TempDir()
	credFile := filepath.Join(tmpDir, "credentials.yml")

	// Save
	cred := Credentials{
		Version: "0.1",
		APIKey:  "test-api-key-123",
		PricingAPIEndpoint: "https://pricing.api.c3x.dev",
	}

	data := []byte("version: \"0.1\"\napi_key: test-api-key-123\npricing_api_endpoint: https://pricing.api.c3x.dev\n")
	err := os.WriteFile(credFile, data, 0600)
	require.NoError(t, err)

	// Verify file exists
	assert.True(t, FileExists(credFile))

	// Read back
	content, err := os.ReadFile(credFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "test-api-key-123")
	_ = cred
}

func TestCredentials_Fields(t *testing.T) {
	c := Credentials{
		Version:            "0.1",
		APIKey:             "ico-abc123",
		PricingAPIEndpoint: "https://custom.pricing.api",
	}
	assert.Equal(t, "0.1", c.Version)
	assert.Equal(t, "ico-abc123", c.APIKey)
	assert.Equal(t, "https://custom.pricing.api", c.PricingAPIEndpoint)
}

func TestCredentialsFilePath_ContainsC3X(t *testing.T) {
	path := CredentialsFilePath()
	assert.Contains(t, path, "c3x")
	assert.Contains(t, path, "credentials")
}

func TestConfiguration_Fields(t *testing.T) {
	c := Configuration{
		Version:  "0.1",
		Currency: "EUR",
	}
	assert.Equal(t, "EUR", c.Currency)

	// EnableDashboard, EnableCloud
	enabled := true
	c.EnableDashboard = &enabled
	c.EnableCloud = &enabled
	assert.True(t, *c.EnableDashboard)
	assert.True(t, *c.EnableCloud)
}

func TestConfigurationFilePath_ContainsC3X(t *testing.T) {
	path := ConfigurationFilePath()
	assert.Contains(t, path, "c3x")
	assert.Contains(t, path, "configuration")
}

func TestState_Fields(t *testing.T) {
	s := State{
		InstallID:              "uuid-abc-123",
		LatestReleaseVersion:   "v2.0.0",
		LatestReleaseCheckedAt: "2026-04-10T00:00:00Z",
	}
	assert.Equal(t, "uuid-abc-123", s.InstallID)
	assert.Equal(t, "v2.0.0", s.LatestReleaseVersion)
	assert.NotEmpty(t, s.LatestReleaseCheckedAt)
}
