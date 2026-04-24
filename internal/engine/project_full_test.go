package engine

import (
	"errors"
	"testing"

	"github.com/c3xdev/c3x/internal/vcs"

	"github.com/stretchr/testify/assert"
)

func TestWorkspace_AddError(t *testing.T) {
	ws := &Workspace{Name: "test", Metadata: &WorkspaceMeta{}}
	diag := NewDiagModuleEvaluationFailure(errors.New("eval"))
	ws.Metadata.AddError(diag)
	assert.True(t, ws.Metadata.HasErrors())
}

func TestWorkspace_HasErrors_NoErrors(t *testing.T) {
	ws := &Workspace{Name: "test", Metadata: &WorkspaceMeta{}}
	assert.False(t, ws.Metadata.HasErrors())
}

func TestWorkspace_IsEmptyProjectError(t *testing.T) {
	ws := &Workspace{Metadata: &WorkspaceMeta{}}
	ws.Metadata.AddError(NewEmptyPathTypeError(errors.New("empty")))
	assert.True(t, ws.Metadata.IsEmptyProjectError())
}

func TestWorkspace_IsRunQuotaExceeded(t *testing.T) {
	ws := &Workspace{Metadata: &WorkspaceMeta{}}
	ws.Metadata.AddError(NewDiagRunQuotaExceeded(errors.New("quota")))
	_, exceeded := ws.Metadata.IsRunQuotaExceeded()
	assert.True(t, exceeded)
}

func TestWorkspace_IsRunQuotaExceeded_False(t *testing.T) {
	ws := &Workspace{Metadata: &WorkspaceMeta{}}
	ws.Metadata.AddError(NewDiagModuleEvaluationFailure(errors.New("mod")))
	_, exceeded := ws.Metadata.IsRunQuotaExceeded()
	assert.False(t, exceeded)
}

func TestWorkspaceMeta_Error(t *testing.T) {
	meta := &WorkspaceMeta{}
	meta.AddError(NewDiagModuleEvaluationFailure(errors.New("eval failed")))
	errMsg := meta.Errors[0].Error()
	assert.NotEmpty(t, errMsg)
}

func TestDiagnostic_Error(t *testing.T) {
	d := &Diagnostic{
		Code:    101,
		Message: "Test error",
		Data:    errors.New("underlying"),
	}
	assert.Contains(t, d.Error(), "Test error")
}

func TestWorkspaceMeta_WorkspaceLabel(t *testing.T) {
	meta := &WorkspaceMeta{
		Path:               "/code/infra",
		TerraformWorkspace: "production",
	}
	label := meta.WorkspaceLabel()
	_ = label // verify no panic
}

func TestWorkspaceMeta_GenerateProjectName(t *testing.T) {
	meta := &WorkspaceMeta{
		Path:               "/code/infra",
		TerraformWorkspace: "default",
	}
	name := meta.GenerateProjectName(vcs.Remote{}, false)
	assert.NotEmpty(t, name)
}

func TestProjects_Len(t *testing.T) {
	ps := Projects{
		{Name: "p1"},
		{Name: "p2"},
	}
	assert.Equal(t, 2, ps.Len())
}

func TestProjects_Swap(t *testing.T) {
	ps := Projects{
		{Name: "p1"},
		{Name: "p2"},
	}
	ps.Swap(0, 1)
	assert.Equal(t, "p2", ps[0].Name)
	assert.Equal(t, "p1", ps[1].Name)
}

func TestDetectProjectMetadata_CurrentDir(t *testing.T) {
	meta := DetectProjectMetadata(".")
	assert.Equal(t, ".", meta.Path)
}

func TestNewProject_FullFields(t *testing.T) {
	meta := &WorkspaceMeta{
		Path:               "/code",
		Type:               "terraform_dir",
		TerraformWorkspace: "production",
		VCSSubPath:         "infra/",
	}
	ws := NewProject("my-project", meta)
	assert.Equal(t, "my-project", ws.Name)
	assert.Equal(t, "/code", ws.Metadata.Path)
	assert.Equal(t, "production", ws.Metadata.TerraformWorkspace)
}

func TestComputeDiff_EmptyBoth(t *testing.T) {
	diff := ComputeDiff(nil, nil)
	assert.Empty(t, diff)
}

func TestDiffName_Various(t *testing.T) {
	assert.Equal(t, "current", diffName("current", ""))
	assert.Equal(t, "past", diffName("", "past"))
	assert.Equal(t, "", diffName("", ""))
	assert.Equal(t, "same", diffName("same", "same"))
	assert.Equal(t, "new", diffName("new", "old"))
}
