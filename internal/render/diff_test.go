package render

import (
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToDiff(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(100),
		DiffTotalMonthlyCost: dp(50),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources:        []Resource{{Name: "r1", MonthlyCost: dp(100)}},
				TotalMonthlyCost: dp(100),
			},
			Diff: &Breakdown{
				Resources:        []Resource{{Name: "r1", MonthlyCost: dp(50)}},
				TotalMonthlyCost: dp(50),
			},
		}},
	}

	output, err := ToDiff(report, Options{NoColor: true})
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestProjectTitle(t *testing.T) {
	p := Project{
		Name:     "my-project",
		Metadata: &engine.WorkspaceMeta{Path: "/code/infra"},
	}
	title := projectTitle(p)
	assert.NotEmpty(t, title)
	assert.Contains(t, title, "my-project")
}

func TestColorizeDiffName(t *testing.T) {
	// Added
	result := colorizeDiffName("+ new-resource")
	assert.Contains(t, result, "new-resource")

	// Removed
	result = colorizeDiffName("- old-resource")
	assert.Contains(t, result, "old-resource")

	// Updated
	result = colorizeDiffName("~ changed-resource")
	assert.Contains(t, result, "changed-resource")
}

func TestZeroDiffComponent(t *testing.T) {
	diff := CostComponent{Name: "cc1", MonthlyCost: dp(0)}
	old := &CostComponent{Name: "cc1", MonthlyCost: dp(100)}
	new := &CostComponent{Name: "cc1", MonthlyCost: dp(100)}

	// Just verify it doesn't panic
	_ = zeroDiffComponent(diff, old, new, "test-resource")
}

func TestZeroDiffResource(t *testing.T) {
	diff := Resource{Name: "r1", MonthlyCost: dp(0)}
	old := &Resource{Name: "r1", MonthlyCost: dp(100)}
	new := &Resource{Name: "r1", MonthlyCost: dp(100)}

	_ = zeroDiffResource(diff, old, new, "r1")
}
