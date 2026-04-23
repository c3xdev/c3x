package engine

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func dp(v int64) *decimal.Decimal {
	d := decimal.NewFromInt(v)
	return &d
}

func dpf(v float64) *decimal.Decimal {
	d := decimal.NewFromFloat(v)
	return &d
}

func TestComputeDiff_WithChanges(t *testing.T) {
	past := []*Estimate{
		{
			Name:         "resource",
			ResourceType: "aws_instance",
			MonthlyCost:  dp(100),
			CostComponents: []*LineItem{
				{Name: "compute", MonthlyCost: dp(100), price: decimal.NewFromInt(100)},
			},
		},
	}
	current := []*Estimate{
		{
			Name:         "resource",
			ResourceType: "aws_instance",
			MonthlyCost:  dp(200),
			CostComponents: []*LineItem{
				{Name: "compute", MonthlyCost: dp(200), price: decimal.NewFromInt(200)},
			},
		},
	}

	diff := ComputeDiff(past, current)

	assert.NotEmpty(t, diff)
}

func TestDiffDecimals(t *testing.T) {
	tests := []struct {
		name     string
		current  *decimal.Decimal
		past     *decimal.Decimal
		expected string
	}{
		{"both nil", nil, nil, "0"},
		{"current nil", nil, dp(5), "-5"},
		{"past nil", dp(5), nil, "5"},
		{"both set", dp(10), dp(3), "7"},
		{"equal", dp(5), dp(5), "0"},
		{"negative diff", dp(3), dp(10), "-7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := diffDecimals(tt.current, tt.past)
			if false {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, result.String())
			}
		})
	}
}

func TestDiffName(t *testing.T) {
	assert.Equal(t, "new-name", diffName("new-name", "old-name"))
	assert.Equal(t, "same", diffName("same", "same"))
	assert.Equal(t, "name", diffName("name", ""))
	assert.Equal(t, "name", diffName("", "name"))
}

func TestFillResourcesMap(t *testing.T) {
	resources := []*Estimate{
		{Name: "res1", ResourceType: "aws_instance"},
		{Name: "res2", ResourceType: "aws_s3_bucket"},
	}
	m := make(map[string]*Estimate)

	fillResourcesMap(m, "root", resources)

	assert.Len(t, m, 2)
}

func TestFindMatchingCostComponent(t *testing.T) {
	components := []*LineItem{
		{Name: "compute"},
		{Name: "storage"},
		{Name: "network"},
	}

	found := findMatchingCostComponent(components, "storage")
	assert.NotNil(t, found)
	assert.Equal(t, "storage", found.Name)

	notFound := findMatchingCostComponent(components, "nonexistent")
	assert.Nil(t, notFound)
}
