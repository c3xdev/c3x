package render

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/settings"
)

func dp(v float64) *decimal.Decimal {
	d := decimal.NewFromFloat(v)
	return &d
}

func intPtr(i int) *int { return &i }

func TestToOutputFormat(t *testing.T) {
	cfg := &settings.Settings{Currency: "USD"}
	ws := &engine.Workspace{
		Name:     "test",
		Metadata: &engine.WorkspaceMeta{Path: "/code", Type: "terraform_dir"},
		Resources: []*engine.Estimate{
			{
				Name:         "aws_instance.web",
				ResourceType: "aws_instance",
				MonthlyCost:  dp(560.64),
				CostComponents: []*engine.LineItem{
					{Name: "Instance usage", Unit: "hours", MonthlyCost: dp(560.64)},
				},
			},
		},
	}

	report, err := ToOutputFormat(cfg, []*engine.Workspace{ws})
	require.NoError(t, err)
	assert.Len(t, report.Projects, 1)
	assert.Equal(t, "test", report.Projects[0].Name)
}

func TestFormatOutput_JSON(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(100),
		Projects: Projects{{
			Name: "test-project",
			Breakdown: &Breakdown{
				Resources:        []Resource{{Name: "r1", MonthlyCost: dp(100)}},
				TotalMonthlyCost: dp(100),
			},
		}},
	}

	output, err := FormatOutput("json", report, Options{NoColor: true})
	require.NoError(t, err)
	assert.Contains(t, string(output), "test-project")
}

func TestFormatOutput_Table(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(560.64),
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

	output, err := FormatOutput("table", report, Options{NoColor: true, Fields: []string{"all"}})
	require.NoError(t, err)
	assert.Contains(t, string(output), "aws_instance.web")
}

func TestFormatOutput_HTML(t *testing.T) {
	report := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(100),
		Projects: Projects{{
			Name:      "test",
			Metadata:  &engine.WorkspaceMeta{Path: "/code"},
			Breakdown: &Breakdown{TotalMonthlyCost: dp(100)},
		}},
	}
	output, err := FormatOutput("html", report, Options{NoColor: true})
	require.NoError(t, err)
	assert.Contains(t, string(output), "html")
}

func TestCheckOutputVersion(t *testing.T) {
	assert.True(t, checkOutputVersion("0.2"))
	assert.False(t, checkOutputVersion("0.1"))
	assert.False(t, checkOutputVersion("1.0"))
	assert.False(t, checkOutputVersion(""))
}

func TestCheckCurrency(t *testing.T) {
	c, err := checkCurrency("", "USD")
	assert.NoError(t, err)
	assert.Equal(t, "USD", c)

	c, err = checkCurrency("USD", "USD")
	assert.NoError(t, err)
	assert.Equal(t, "USD", c)

	_, err = checkCurrency("USD", "EUR")
	assert.Error(t, err)
}

func TestNewMetadata(t *testing.T) {
	ctx, _ := settings.NewRunContextFromEnv(context.TODO())
	if ctx == nil {
		t.Skip("RunContext creation requires environment setup")
		return
	}
	ctx.CMD = "estimate"
	meta := NewMetadata(ctx)
	assert.Equal(t, "estimate", meta.C3XCommand)
}

func TestReport_Fields(t *testing.T) {
	r := Report{
		Version:          outputVersion,
		Currency:         "USD",
		TotalMonthlyCost: dp(100),
		Summary:          &Summary{TotalDetectedResources: intPtr(5)},
	}
	assert.Equal(t, "0.2", r.Version)
	assert.Equal(t, "USD", r.Currency)
	assert.Equal(t, 5, *r.Summary.TotalDetectedResources)
}

func TestCombine_SingleInput(t *testing.T) {
	input := ReportInput{
		Metadata: map[string]string{"path": "test.json"},
		Root: Report{
			Version:          outputVersion,
			Currency:         "USD",
			TotalMonthlyCost: dp(100),
			Projects: Projects{{
				Name:      "proj1",
				Breakdown: &Breakdown{TotalMonthlyCost: dp(100)},
			}},
		},
	}

	combined, err := Combine([]ReportInput{input})
	require.NoError(t, err)
	assert.Len(t, combined.Projects, 1)
}

func TestFindResourceByName(t *testing.T) {
	resources := []Resource{{Name: "r1"}, {Name: "r2"}, {Name: "r3"}}

	found := findResourceByName(resources, "r2")
	assert.NotNil(t, found)
	assert.Equal(t, "r2", found.Name)

	assert.Nil(t, findResourceByName(resources, "r99"))
}

func TestFindMatchingCostComponent_Render(t *testing.T) {
	components := []CostComponent{{Name: "compute"}, {Name: "storage"}}

	found := findMatchingCostComponent(components, "storage")
	assert.NotNil(t, found)
	assert.Equal(t, "storage", found.Name)

	assert.Nil(t, findMatchingCostComponent(components, "nonexistent"))
}

func TestOpChar(t *testing.T) {
	assert.Contains(t, opChar(ADDED), "+")
	assert.Contains(t, opChar(REMOVED), "-")
	assert.Contains(t, opChar(UPDATED), "~")
}
