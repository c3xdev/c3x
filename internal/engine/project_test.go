package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewProject(t *testing.T) {
	meta := &WorkspaceMeta{Path: "/code", Type: "terraform_dir"}
	ws := NewProject("test-project", meta)

	assert.Equal(t, "test-project", ws.Name)
	assert.Equal(t, "/code", ws.Metadata.Path)
}

func TestWorkspace_AllResources(t *testing.T) {
	ws := &Workspace{
		Resources: []*Estimate{{Name: "r1"}, {Name: "r2"}},
	}
	assert.Len(t, ws.AllResources(), 2)
}

func TestIsEmptyPathTypeError(t *testing.T) {
	err := NewEmptyPathTypeError(errors.New("test"))
	assert.True(t, IsEmptyPathTypeError(err))
	assert.False(t, IsEmptyPathTypeError(errors.New("other")))
}

func TestNewDiagnosticConstructors(t *testing.T) {
	d := NewDiagModuleEvaluationFailure(errors.New("mod"))
	assert.Equal(t, 102, d.Code)
	assert.NotEmpty(t, d.Message)

	d2 := NewDiagJSONParsingFailure(errors.New("json"))
	assert.Equal(t, 101, d2.Code)
}

func TestDetectProjectMetadata(t *testing.T) {
	meta := DetectProjectMetadata("/some/path")
	assert.Equal(t, "/some/path", meta.Path)
}
