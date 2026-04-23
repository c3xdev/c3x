package render

import (
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToTable(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(742.64),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources: []Resource{
					{
						Name:         "aws_instance.web",
						ResourceType: "aws_instance",
						MonthlyCost:  dp(560.64),
						CostComponents: []CostComponent{
							{Name: "Instance usage", Unit: "hours", MonthlyCost: dp(560.64)},
						},
					},
					{
						Name:         "aws_lambda_function.api",
						ResourceType: "aws_lambda_function",
						MonthlyCost:  dp(0),
						CostComponents: []CostComponent{
							{Name: "Requests", Unit: "1M requests", UsageBased: true},
						},
					},
				},
				TotalMonthlyCost: dp(560.64),
			},
		}},
		Summary: &Summary{
			TotalDetectedResources: intPtr(2),
			TotalSupportedResources: intPtr(2),
		},
	}

	opts := Options{
		NoColor: true,
		Fields:  []string{"all"},
	}

	output, err := ToTable(report, opts)
	require.NoError(t, err)

	result := string(output)
	assert.Contains(t, result, "aws_instance.web")
	assert.Contains(t, result, "Instance usage")
	assert.Contains(t, result, "561")
}

func TestToTable_EmptyProject(t *testing.T) {
	report := Report{
		Version:  outputVersion,
		Currency: "USD",
		Projects: Projects{{
			Name:     "empty",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources:        []Resource{},
				TotalMonthlyCost: dp(0),
			},
		}},
		TotalMonthlyCost: dp(0),
	}

	output, err := ToTable(report, Options{NoColor: true})
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestToTable_WithSkippedResources(t *testing.T) {
	report := Report{
		Version:  outputVersion,
		Currency: "USD",
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources: []Resource{
					{Name: "aws_instance.web", MonthlyCost: dp(100)},
				},
				FreeResources: []Resource{
					{Name: "aws_vpc.main"},
				},
				TotalMonthlyCost: dp(100),
			},
		}},
		TotalMonthlyCost: dp(100),
		Summary: &Summary{
			TotalDetectedResources: intPtr(2),
			TotalNoPriceResources:     intPtr(1),
		},
	}

	opts := Options{NoColor: true, ShowSkipped: true}
	output, err := ToTable(report, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestToTable_MultiProject(t *testing.T) {
	report := Report{
		Version:  outputVersion,
		Currency: "USD",
		Projects: Projects{
			{
				Name:     "proj1",
				Metadata: &engine.WorkspaceMeta{Path: "/code/proj1"},
				Breakdown: &Breakdown{
					Resources:        []Resource{{Name: "r1", MonthlyCost: dp(100)}},
					TotalMonthlyCost: dp(100),
				},
			},
			{
				Name:     "proj2",
				Metadata: &engine.WorkspaceMeta{Path: "/code/proj2"},
				Breakdown: &Breakdown{
					Resources:        []Resource{{Name: "r2", MonthlyCost: dp(200)}},
					TotalMonthlyCost: dp(200),
				},
			},
		},
		TotalMonthlyCost: dp(300),
	}

	output, err := ToTable(report, Options{NoColor: true})
	require.NoError(t, err)
	assert.Contains(t, string(output), "proj1")
	assert.Contains(t, string(output), "proj2")
}
