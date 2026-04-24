package render

import (
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToMarkdown(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(742.64),
		Projects: Projects{{
			Name:     "test",
			Metadata: &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{
				Resources: []Resource{{
					Name:        "aws_instance.web",
					MonthlyCost: dp(560.64),
				}},
				TotalMonthlyCost: dp(560.64),
			},
		}},
	}

	opts := Options{NoColor: true}
	mdOpts := MarkdownOptions{}

	output, err := ToMarkdown(report, opts, mdOpts)
	require.NoError(t, err)
	assert.NotEmpty(t, output.Msg)
}

func TestToSlackMessage(t *testing.T) {
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

	output, err := ToSlackMessage(report, Options{NoColor: true})
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}
