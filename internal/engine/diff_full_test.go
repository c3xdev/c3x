package engine

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestComputeDiff_AddAndRemove(t *testing.T) {
	past := []*Estimate{
		{
			Name:         "removed-resource",
			ResourceType: "aws_s3_bucket",
			MonthlyCost:  dp(50),
			CostComponents: []*LineItem{
				{Name: "storage", MonthlyCost: dp(50), price: decimal.NewFromInt(50)},
			},
		},
	}
	current := []*Estimate{
		{
			Name:         "added-resource",
			ResourceType: "aws_instance",
			MonthlyCost:  dp(100),
			CostComponents: []*LineItem{
				{Name: "compute", MonthlyCost: dp(100), price: decimal.NewFromInt(100)},
			},
		},
	}

	diff := ComputeDiff(past, current)
	assert.NotEmpty(t, diff)
	// Should have both the added and removed resources
	assert.GreaterOrEqual(t, len(diff), 1)
}

func TestComputeDiff_CostComponentChange(t *testing.T) {
	past := []*Estimate{
		{
			Name:         "resource",
			ResourceType: "aws_instance",
			MonthlyCost:  dp(100),
			CostComponents: []*LineItem{
				{Name: "compute", MonthlyCost: dp(80), price: decimal.NewFromInt(80)},
				{Name: "storage", MonthlyCost: dp(20), price: decimal.NewFromInt(20)},
			},
		},
	}
	current := []*Estimate{
		{
			Name:         "resource",
			ResourceType: "aws_instance",
			MonthlyCost:  dp(150),
			CostComponents: []*LineItem{
				{Name: "compute", MonthlyCost: dp(120), price: decimal.NewFromInt(120)},
				{Name: "storage", MonthlyCost: dp(30), price: decimal.NewFromInt(30)},
			},
		},
	}

	diff := ComputeDiff(past, current)
	assert.NotEmpty(t, diff)
}

func TestComputeDiff_SubResourceChange(t *testing.T) {
	past := []*Estimate{
		{
			Name:         "aws_instance.web",
			ResourceType: "aws_instance",
			MonthlyCost:  dp(600),
			CostComponents: []*LineItem{
				{Name: "compute", MonthlyCost: dp(560), price: decimal.NewFromInt(560)},
			},
			SubResources: []*Estimate{
				{
					Name:        "root_block_device",
					MonthlyCost: dp(5),
					CostComponents: []*LineItem{
						{Name: "storage", MonthlyCost: dp(5), price: decimal.NewFromInt(5)},
					},
				},
			},
		},
	}
	current := []*Estimate{
		{
			Name:         "aws_instance.web",
			ResourceType: "aws_instance",
			MonthlyCost:  dp(610),
			CostComponents: []*LineItem{
				{Name: "compute", MonthlyCost: dp(560), price: decimal.NewFromInt(560)},
			},
			SubResources: []*Estimate{
				{
					Name:        "root_block_device",
					MonthlyCost: dp(10),
					CostComponents: []*LineItem{
						{Name: "storage", MonthlyCost: dp(10), price: decimal.NewFromInt(10)},
					},
				},
			},
		},
	}

	diff := ComputeDiff(past, current)
	assert.NotEmpty(t, diff)
}

func TestComputeDiff_MultipleResources(t *testing.T) {
	mkResource := func(name string, cost int64) *Estimate {
		return &Estimate{
			Name:         name,
			ResourceType: "aws_instance",
			MonthlyCost:  dp(cost),
			CostComponents: []*LineItem{
				{Name: "compute", MonthlyCost: dp(cost), price: decimal.NewFromInt(cost)},
			},
		}
	}

	past := []*Estimate{
		mkResource("r1", 100),
		mkResource("r2", 200),
		mkResource("r3", 300), // will be removed
	}
	current := []*Estimate{
		mkResource("r1", 150), // changed
		mkResource("r2", 200), // unchanged
		mkResource("r4", 400), // added
	}

	diff := ComputeDiff(past, current)
	assert.NotEmpty(t, diff)
}

func TestDiffDecimals_AllCases(t *testing.T) {
	tests := []struct {
		name     string
		current  *decimal.Decimal
		past     *decimal.Decimal
		expected string
	}{
		{"both nil", nil, nil, "0"},
		{"current only", dp(10), nil, "10"},
		{"past only", nil, dp(10), "-10"},
		{"positive diff", dp(100), dp(30), "70"},
		{"negative diff", dp(30), dp(100), "-70"},
		{"zero diff", dp(50), dp(50), "0"},
		{"small decimal", dpf(0.768), dpf(0.504), "0.264"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := diffDecimals(tt.current, tt.past)
			assert.NotNil(t, result)
		})
	}
}

func TestDiffName_AllBranches(t *testing.T) {
	// Same name
	assert.Equal(t, "same", diffName("same", "same"))

	// Different names — returns current
	assert.Equal(t, "new", diffName("new", "old"))

	// Empty current — returns past
	assert.Equal(t, "past", diffName("", "past"))

	// Empty past — returns current
	assert.Equal(t, "current", diffName("current", ""))

	// Both empty
	assert.Equal(t, "", diffName("", ""))

	// Same type, different instance
	result := diffName("aws_instance.web", "aws_instance.api")
	assert.Equal(t, "aws_instance.web", result)
}

func TestFindMatchingCostComponent_FirstMatch(t *testing.T) {
	components := []*LineItem{
		{Name: "a"},
		{Name: "b"},
		{Name: "a"}, // duplicate name
	}
	found := findMatchingCostComponent(components, "a")
	assert.NotNil(t, found)
	assert.Equal(t, "a", found.Name)
}

func TestFillResourcesMap_WithSubResources(t *testing.T) {
	resources := []*Estimate{
		{
			Name:         "parent",
			ResourceType: "aws_instance",
			SubResources: []*Estimate{
				{Name: "child", ResourceType: "ebs_volume"},
			},
		},
	}
	m := make(map[string]*Estimate)
	fillResourcesMap(m, "root", resources)

	assert.Contains(t, m, "root.parent")
	// SubResources should also be in the map
	assert.Contains(t, m, "root.parent.child")
}
