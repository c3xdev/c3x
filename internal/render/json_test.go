package render

import (
	"encoding/json"
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
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
				},
				TotalMonthlyCost: dp(560.64),
			},
		}},
	}

	output, err := ToJSON(report, Options{NoColor: true})
	require.NoError(t, err)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal(output, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "0.2", parsed["version"])
	assert.Equal(t, "USD", parsed["currency"])

	projects := parsed["projects"].([]interface{})
	assert.Len(t, projects, 1)
}

func TestToJSON_EmptyProjects(t *testing.T) {
	report := Report{
		Version:  outputVersion,
		Currency: "USD",
		Projects: Projects{},
	}

	output, err := ToJSON(report, Options{})
	require.NoError(t, err)
	assert.Contains(t, string(output), `"projects"`)
}
