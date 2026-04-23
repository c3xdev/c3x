package render

import (
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombine_MultipleInputs(t *testing.T) {
	input1 := ReportInput{
		Metadata: map[string]string{"path": "proj1.json"},
		Root: Report{
			Version:          outputVersion,
			Currency:         "USD",
			TotalMonthlyCost: dp(100),
			Projects: Projects{{
				Name:     "proj1",
				Metadata: &engine.WorkspaceMeta{Path: "/code/proj1"},
				Breakdown: &Breakdown{
					Resources:        []Resource{{Name: "r1", MonthlyCost: dp(100)}},
					TotalMonthlyCost: dp(100),
				},
			}},
		},
	}

	input2 := ReportInput{
		Metadata: map[string]string{"path": "proj2.json"},
		Root: Report{
			Version:          outputVersion,
			Currency:         "USD",
			TotalMonthlyCost: dp(200),
			Projects: Projects{{
				Name:     "proj2",
				Metadata: &engine.WorkspaceMeta{Path: "/code/proj2"},
				Breakdown: &Breakdown{
					Resources:        []Resource{{Name: "r2", MonthlyCost: dp(200)}},
					TotalMonthlyCost: dp(200),
				},
			}},
		},
	}

	combined, err := Combine([]ReportInput{input1, input2})
	require.NoError(t, err)
	assert.Len(t, combined.Projects, 2)
	assert.Equal(t, "300", combined.TotalMonthlyCost.StringFixed(0))
}

func TestCombine_EmptyInputs(t *testing.T) {
	combined, err := Combine([]ReportInput{})
	require.NoError(t, err)
	assert.Empty(t, combined.Projects)
}

func TestCombine_CurrencyMismatch(t *testing.T) {
	input1 := ReportInput{
		Root: Report{Version: outputVersion, Currency: "USD", Projects: Projects{}},
	}
	input2 := ReportInput{
		Root: Report{Version: outputVersion, Currency: "EUR", Projects: Projects{}},
	}

	_, err := Combine([]ReportInput{input1, input2})
	assert.Error(t, err)
}

func TestFormatOutput_AllFormats(t *testing.T) {
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

	formats := []string{"json", "table", "html"}
	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			output, err := FormatOutput(format, report, Options{NoColor: true, Fields: []string{"all"}})
			require.NoError(t, err)
			assert.NotEmpty(t, output)
		})
	}
}

func TestFormatOutput_SlackMessage(t *testing.T) {
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

	output, err := FormatOutput("slack-message", report, Options{NoColor: true})
	require.NoError(t, err)
	assert.NotEmpty(t, output)
}
