package settings

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession_FullLifecycle(t *testing.T) {
	ctx, err := NewRunContextFromEnv(context.Background())
	require.NoError(t, err)

	// UUID
	assert.NotEmpty(t, ctx.UUID())

	// StartTime
	assert.Greater(t, ctx.StartTime, int64(0))

	// Context
	assert.NotNil(t, ctx.Context())

	// Config
	assert.NotNil(t, ctx.Config)

	// ContextValues
	assert.NotNil(t, ctx.ContextValues)

	// SetIsC3XComment
	ctx.SetIsC3XComment()
	assert.True(t, ctx.IsC3XComment())

	// IsAutoDetect
	assert.True(t, ctx.IsAutoDetect())

	// IsCIRun
	_ = ctx.IsCIRun()

	// GetParallelism
	p, _ := ctx.GetParallelism()
	assert.GreaterOrEqual(t, p, 4)

	// IsCloudEnabled
	_ = ctx.IsCloudEnabled()

	// IsCloudUploadEnabled
	_ = ctx.IsCloudUploadEnabled()
}

func TestContextValues_ThreadSafety(t *testing.T) {
	cv := NewContextValues(map[string]interface{}{})

	// Concurrent writes shouldn't panic
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			cv.SetValue("key", n)
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	val, ok := cv.GetValue("key")
	assert.True(t, ok)
	assert.NotNil(t, val)
}

func TestSettings_LoadFromEnv_AllKeys(t *testing.T) {
	envVars := map[string]string{
		"C3X_API_KEY":              "test-key",
		"C3X_PRICING_API_ENDPOINT": "https://test.api",
		"C3X_LOG_LEVEL":            "debug",
		"C3X_NO_COLOR":             "true",
		"C3X_SKIP_UPDATE_CHECK":    "true",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	cfg := DefaultConfig()
	err := cfg.LoadFromEnv()
	require.NoError(t, err)

	assert.Equal(t, "test-key", cfg.APIKey)
	assert.Equal(t, "https://test.api", cfg.PricingAPIEndpoint)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.True(t, cfg.NoColor)
	assert.True(t, cfg.SkipUpdateCheck)
}

func TestSettings_WriteLevel_AllLevels(t *testing.T) {
	levels := []string{"trace", "debug", "info", "warn", "error"}
	for _, level := range levels {
		cfg := DefaultConfig()
		cfg.LogLevel = level
		assert.Equal(t, level, cfg.WriteLevel())
	}
}

func TestSettings_WorkingDirectory_Variants(t *testing.T) {
	// With RootPath
	cfg := DefaultConfig()
	cfg.RootPath = "/tmp/test"
	assert.Equal(t, "/tmp/test", cfg.WorkingDirectory())

	// With ConfigFilePath
	cfg2 := DefaultConfig()
	cfg2.ConfigFilePath = "/code/c3x.yml"
	wd := cfg2.WorkingDirectory()
	assert.NotEmpty(t, wd)
}

func TestSession_EventEnvWithProjectContexts(t *testing.T) {
	ctx, _ := NewRunContextFromEnv(context.Background())
	env := ctx.EventEnvWithProjectContexts(nil)
	assert.NotNil(t, env)
	// Should contain basic info
}

func TestEmptyRunContext_Fields(t *testing.T) {
	ctx := EmptyRunContext()
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.Config)
	assert.NotNil(t, ctx.ContextValues)
}
