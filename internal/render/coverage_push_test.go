package render

import (
	"testing"

	"github.com/c3xdev/c3x/internal/settings"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatOutput_Diff(t *testing.T) {
	report := Report{
		Version:              outputVersion,
		Currency:             "USD",
		TotalMonthlyCost:     dp(200),
		DiffTotalMonthlyCost: dp(100),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources:        []Resource{{Name: "r1", MonthlyCost: dp(200)}},
				TotalMonthlyCost: dp(200),
			},
			Diff: &Breakdown{
				Resources:        []Resource{{Name: "r1", MonthlyCost: dp(100)}},
				TotalMonthlyCost: dp(100),
			},
		}},
	}

	output, err := FormatOutput("diff", report, Options{NoColor: true})
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestFormatOutput_GithubComment(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(100),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources:        []Resource{{Name: "r1", MonthlyCost: dp(100)}},
				TotalMonthlyCost: dp(100),
			},
		}},
	}

	output, err := FormatOutput("github-comment", report, Options{NoColor: true})
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestFormatOutput_GitlabComment(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(100),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources:        []Resource{{Name: "r1", MonthlyCost: dp(100)}},
				TotalMonthlyCost: dp(100),
			},
		}},
	}

	output, err := FormatOutput("gitlab-comment", report, Options{NoColor: true})
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestFormatOutput_AzureReposComment(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(100),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				TotalMonthlyCost: dp(100),
			},
		}},
	}

	output, err := FormatOutput("azure-repos-comment", report, Options{NoColor: true})
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestFormatOutput_BitbucketComment(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(100),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				TotalMonthlyCost: dp(100),
			},
		}},
	}

	output, err := FormatOutput("bitbucket-comment", report, Options{NoColor: true})
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestCompareTo(t *testing.T) {
	cfg := &settings.Settings{Currency: "USD"}
	current := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(200),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources: []Resource{
					{Name: "r1", MonthlyCost: dp(200), CostComponents: []CostComponent{{Name: "cc1", MonthlyCost: dp(200)}}},
				},
				TotalMonthlyCost: dp(200),
			},
		}},
	}
	prior := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(100),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources: []Resource{
					{Name: "r1", MonthlyCost: dp(100), CostComponents: []CostComponent{{Name: "cc1", MonthlyCost: dp(100)}}},
				},
				TotalMonthlyCost: dp(100),
			},
		}},
	}

	result, err := CompareTo(cfg, current, prior)
	require.NoError(t, err)
	assert.NotNil(t, result.DiffTotalMonthlyCost)
}

