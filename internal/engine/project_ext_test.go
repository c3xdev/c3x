package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDiagTerragruntModuleEvaluationFailure(t *testing.T) {
	d := NewDiagTerragruntModuleEvaluationFailure(errors.New("eval failed"))
	assert.Equal(t, 104, d.Code)
	assert.Contains(t, d.Message, "eval failed")
}

func TestNewDiagTerragruntEvaluationFailure(t *testing.T) {
	d := NewDiagTerragruntEvaluationFailure(errors.New("tg error"))
	assert.Equal(t, 103, d.Code)
}

func TestNewDiagRunQuotaExceeded(t *testing.T) {
	d := NewDiagRunQuotaExceeded(errors.New("quota"))
	assert.Equal(t, 401, d.Code)
}

func TestNewDiagMissingVars(t *testing.T) {
	d := NewDiagMissingVars("var1", "var2", "var3")
	assert.Equal(t, 105, d.Code)
	assert.NotEmpty(t, d.Message)
}

func TestNewFailedDownloadDiagnostic(t *testing.T) {
	d := NewFailedDownloadDiagnostic("git::https://example.com/module", errors.New("timeout"))
	assert.NotNil(t, d)
	assert.Contains(t, d.Message, "timeout")
}

func TestNewPrivateRegistryDiag(t *testing.T) {
	loc := "us-east-1"
	d := NewPrivateRegistryDiag("registry.terraform.io/hashicorp/aws", &loc, errors.New("auth failed"))
	assert.NotNil(t, d)
}

func TestNewEmptyPathTypeError(t *testing.T) {
	err := NewEmptyPathTypeError(errors.New("empty path"))
	assert.NotNil(t, err)
	assert.True(t, IsEmptyPathTypeError(err))
}

func TestWorkspace_HasDiffField(t *testing.T) {
	ws := &Workspace{HasDiff: true}
	assert.True(t, ws.HasDiff)
}

func TestWorkspace_DisplayName(t *testing.T) {
	ws := &Workspace{
		Name:     "test-project",
		Metadata: &WorkspaceMeta{Path: "/code/infra"},
	}
	assert.Equal(t, "test-project", ws.Name)
	assert.Equal(t, "/code/infra", ws.Metadata.Path)
}

func TestWorkspaceMeta_Fields(t *testing.T) {
	meta := &WorkspaceMeta{
		Path:                "/code",
		Type:                "terraform_dir",
		TerraformModulePath: "modules/vpc",
		TerraformWorkspace:  "production",
	}
	assert.Equal(t, "/code", meta.Path)
	assert.Equal(t, "terraform_dir", meta.Type)
	assert.Equal(t, "modules/vpc", meta.TerraformModulePath)
	assert.Equal(t, "production", meta.TerraformWorkspace)
}

func TestDiagnostic_Fields(t *testing.T) {
	d := &Diagnostic{
		Code:    101,
		Message: "test error",
		Data:    errors.New("underlying"),
	}
	assert.Equal(t, 101, d.Code)
	assert.Equal(t, "test error", d.Message)
	assert.NotNil(t, d.Data)
}
