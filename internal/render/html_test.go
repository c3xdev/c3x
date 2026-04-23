package render

import (
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToHTML(t *testing.T) {
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
					CostComponents: []CostComponent{
						{Name: "Instance usage", Unit: "hours", MonthlyCost: dp(560.64)},
					},
				}},
				TotalMonthlyCost: dp(560.64),
			},
		}},
	}

	output, err := ToHTML(report, Options{NoColor: true})
	require.NoError(t, err)
	assert.Contains(t, string(output), "<html")
	assert.Contains(t, string(output), "aws_instance.web")
}
