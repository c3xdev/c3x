package render

import (
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToOutputFormat_MultipleProjects(t *testing.T) {
	cfg := &settings.Settings{Currency: "USD"}

	p1 := &engine.Workspace{
		Name:     "proj1",
		Metadata: &engine.WorkspaceMeta{Path: "/code/proj1"},
		Resources: []*engine.Estimate{
			{Name: "r1", ResourceType: "aws_instance", MonthlyCost: dp(100)},
		},
	}
	p2 := &engine.Workspace{
		Name:     "proj2",
		Metadata: &engine.WorkspaceMeta{Path: "/code/proj2"},
		Resources: []*engine.Estimate{
			{Name: "r2", ResourceType: "aws_s3_bucket", MonthlyCost: dp(50)},
		},
	}

	report, err := ToOutputFormat(cfg, []*engine.Workspace{p1, p2})
	require.NoError(t, err)
	assert.Len(t, report.Projects, 2)
}

func TestToOutputFormat_EmptyProjects(t *testing.T) {
	cfg := &settings.Settings{Currency: "USD"}
	report, err := ToOutputFormat(cfg, []*engine.Workspace{})
	require.NoError(t, err)
	assert.Empty(t, report.Projects)
}

func TestToOutputFormat_WithSubResources(t *testing.T) {
	cfg := &settings.Settings{Currency: "USD"}

	p := &engine.Workspace{
		Name:     "test",
		Metadata: &engine.WorkspaceMeta{Path: "/code"},
		Resources: []*engine.Estimate{
			{
				Name:         "aws_instance.web",
				ResourceType: "aws_instance",
				MonthlyCost:  dp(600),
				CostComponents: []*engine.LineItem{
					{Name: "Instance usage", Unit: "hours", MonthlyCost: dp(560)},
				},
				SubResources: []*engine.Estimate{
					{
						Name:        "root_block_device",
						MonthlyCost: dp(5),
						CostComponents: []*engine.LineItem{
							{Name: "Storage", Unit: "GB", MonthlyCost: dp(5)},
						},
					},
				},
			},
		},
	}

	report, err := ToOutputFormat(cfg, []*engine.Workspace{p})
	require.NoError(t, err)
	assert.Len(t, report.Projects[0].Breakdown.Resources, 1)
	assert.Len(t, report.Projects[0].Breakdown.Resources[0].SubResources, 1)
}

func TestToOutputFormat_WithDiff(t *testing.T) {
	cfg := &settings.Settings{Currency: "USD"}

	p := &engine.Workspace{
		Name:     "test",
		Metadata: &engine.WorkspaceMeta{Path: "/code"},
		Resources: []*engine.Estimate{
			{Name: "r1", ResourceType: "aws_instance", MonthlyCost: dp(200),
				CostComponents: []*engine.LineItem{
					{Name: "compute", MonthlyCost: dp(200)},
				}},
		},
		PastResources: []*engine.Estimate{
			{Name: "r1", ResourceType: "aws_instance", MonthlyCost: dp(100),
				CostComponents: []*engine.LineItem{
					{Name: "compute", MonthlyCost: dp(100)},
				}},
		},
		HasDiff: true,
		Diff: []*engine.Estimate{
			{Name: "r1", ResourceType: "aws_instance", MonthlyCost: dp(100)},
		},
	}

	report, err := ToOutputFormat(cfg, []*engine.Workspace{p})
	require.NoError(t, err)
	assert.NotNil(t, report.Projects[0].Diff)
}

func TestResource_Fields(t *testing.T) {
	r := Resource{
		Name:         "aws_instance.web",
		ResourceType: "aws_instance",
		MonthlyCost:  dp(560.64),
		Tags:         &map[string]string{"env": "prod"},
		CostComponents: []CostComponent{
			{Name: "Instance usage", Unit: "hours", MonthlyCost: dp(560.64)},
		},
	}
	assert.Equal(t, "aws_instance.web", r.Name)
	assert.Equal(t, "aws_instance", r.ResourceType)
	assert.Len(t, r.CostComponents, 1)
	assert.Equal(t, "prod", (*r.Tags)["env"])
}

func TestCostComponent_Fields(t *testing.T) {
	price := decimal.NewFromFloat(0.768)
	cc := CostComponent{
		Name:            "Instance usage",
		Unit:            "hours",
		HourlyQuantity:  dp(1),
		MonthlyQuantity: dp(730),
		Price:           price,
		HourlyCost:      dp(0.768),
		MonthlyCost:     dp(560.64),
		UsageBased:      false,
		PriceNotFound:   false,
	}
	assert.Equal(t, "Instance usage", cc.Name)
	assert.Equal(t, "hours", cc.Unit)
	assert.False(t, cc.UsageBased)
	assert.False(t, cc.PriceNotFound)
}

func TestBreakdown_Fields(t *testing.T) {
	b := Breakdown{
		Resources: []Resource{
			{Name: "r1", MonthlyCost: dp(100)},
			{Name: "r2", MonthlyCost: dp(200)},
		},
		TotalHourlyCost:  dp(0.411),
		TotalMonthlyCost: dp(300),
	}
	assert.Len(t, b.Resources, 2)
	assert.Equal(t, "300", b.TotalMonthlyCost.StringFixed(0))
}
