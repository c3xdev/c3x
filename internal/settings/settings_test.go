package settings

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.DefaultPricingAPIEndpoint)
	assert.Equal(t, "table", cfg.Format)
}

func TestSettings_WorkingDirectory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RootPath = "/tmp/test"

	wd := cfg.WorkingDirectory()
	assert.Equal(t, "/tmp/test", wd)
}

func TestIsTest(t *testing.T) {
	// Set test env
	os.Setenv("C3X_ENV", "test")
	defer os.Unsetenv("C3X_ENV")

	assert.True(t, IsTest())
}

func TestIsDev(t *testing.T) {
	os.Setenv("C3X_ENV", "dev")
	defer os.Unsetenv("C3X_ENV")

	assert.True(t, IsDev())
}

func TestCleanProjectName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-project", "my-project"},
		{"My Project", "My Project"},
		{"", ""},
	}

	for _, tt := range tests {
		result := CleanProjectName(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestRootDir(t *testing.T) {
	dir := RootDir()
	assert.NotEmpty(t, dir)
	// Should point to the project root
	assert.Contains(t, dir, "c3x")
}

func TestIsEnvPresent(t *testing.T) {
	os.Setenv("C3X_TEST_VAR", "value")
	defer os.Unsetenv("C3X_TEST_VAR")

	assert.True(t, IsEnvPresent("C3X_TEST_VAR"))
	assert.False(t, IsEnvPresent("C3X_NONEXISTENT_VAR"))
}

func TestFileExists(t *testing.T) {
	// Create a temp file
	f, err := os.CreateTemp("", "c3x-test-*")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	f.Close()

	assert.True(t, FileExists(f.Name()))
	assert.False(t, FileExists("/nonexistent/path/file.txt"))
}

func TestNewContextValues(t *testing.T) {
	cv := NewContextValues(map[string]interface{}{
		"key1": "val1",
		"key2": 42,
	})

	assert.NotNil(t, cv)
}

func TestContextValues_SetGetValue(t *testing.T) {
	cv := NewContextValues(map[string]interface{}{})

	cv.SetValue("test-key", "test-value")

	val, ok := cv.GetValue("test-key")
	assert.True(t, ok)
	assert.Equal(t, "test-value", val)

	_, ok = cv.GetValue("missing-key")
	assert.False(t, ok)
}

func TestSettings_C3XDir(t *testing.T) {
	assert.Equal(t, ".c3x", C3XDir)
}

