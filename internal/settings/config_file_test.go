package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "c3x.yml")
	err := os.WriteFile(configPath, []byte(`version: "0.1"
projects:
  - path: .
    name: test-project
`), 0644)
	require.NoError(t, err)

	spec, err := LoadConfigFile(configPath)
	require.NoError(t, err)
	assert.Len(t, spec.Projects, 1)
	assert.Equal(t, "test-project", spec.Projects[0].Name)
}

func TestLoadConfigFile_InvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "c3x.yml")
	err := os.WriteFile(configPath, []byte(`version: "99.99"
projects: []
`), 0644)
	require.NoError(t, err)

	_, err = LoadConfigFile(configPath)
	assert.Error(t, err)
}

func TestLoadConfigFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "c3x.yml")
	err := os.WriteFile(configPath, []byte(`invalid: [yaml: {broken`), 0644)
	require.NoError(t, err)

	_, err = LoadConfigFile(configPath)
	assert.Error(t, err)
}

func TestLoadConfigFile_NotFound(t *testing.T) {
	_, err := LoadConfigFile("/nonexistent/c3x.yml")
	assert.Error(t, err)
}

func TestLoadConfigFile_MultipleProjects(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "c3x.yml")
	err := os.WriteFile(configPath, []byte(`version: "0.1"
projects:
  - path: ./proj1
    name: project-1
  - path: ./proj2
    name: project-2
`), 0644)
	require.NoError(t, err)

	spec, err := LoadConfigFile(configPath)
	require.NoError(t, err)
	assert.Len(t, spec.Projects, 2)
}

func TestLoadConfigFile_WithTerraformVars(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "c3x.yml")
	err := os.WriteFile(configPath, []byte(`version: "0.1"
projects:
  - path: .
    terraform_var_files:
      - prod.tfvars
    terraform_vars:
      region: us-east-1
`), 0644)
	require.NoError(t, err)

	spec, err := LoadConfigFile(configPath)
	require.NoError(t, err)
	assert.Len(t, spec.Projects[0].TerraformVarFiles, 1)
}

func TestCheckVersion(t *testing.T) {
	assert.True(t, checkVersion("0.1"))
}

func TestCheckVersion_Invalid(t *testing.T) {
	assert.False(t, checkVersion("99.99"))
}

func TestCheckVersion_Empty(t *testing.T) {
	assert.False(t, checkVersion(""))
}

func TestYamlError_AddAndIsValid(t *testing.T) {
	ye := &YamlError{base: "config validation"}

	assert.False(t, ye.isValid()) // No errors yet

	ye.add(assert.AnError)
	assert.True(t, ye.isValid()) // Has errors now
	assert.Contains(t, ye.Error(), "config validation")
}

func TestSettingsFileSpec_UnmarshalYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "c3x.yml")
	err := os.WriteFile(configPath, []byte(`version: "0.1"
projects:
  - path: .
    name: test
    terraform_workspace: production
`), 0644)
	require.NoError(t, err)

	spec, err := LoadConfigFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "production", spec.Projects[0].TerraformWorkspace)
}


