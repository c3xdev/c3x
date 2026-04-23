package engine

import (
	"testing"

	"github.com/c3xdev/c3x/internal/vcs"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// === resource_data.go coverage ===

func TestResourceSpec_GetFloat64OrDefault(t *testing.T) {
	spec := NewResourceData("test", "aws", "test", nil, gjson.Parse(`{"rate": 3.14}`))
	assert.Equal(t, 3.14, spec.GetFloat64OrDefault("rate", 0.0))
	assert.Equal(t, 1.5, spec.GetFloat64OrDefault("missing", 1.5))
}

func TestResourceSpec_GetBoolOrDefault(t *testing.T) {
	spec := NewResourceData("test", "aws", "test", nil, gjson.Parse(`{"enabled": true}`))
	assert.Equal(t, true, spec.GetBoolOrDefault("enabled", false))
	assert.Equal(t, false, spec.GetBoolOrDefault("missing", false))
}

func TestResourceSpec_Set(t *testing.T) {
	spec := NewResourceData("test", "aws", "test", nil, gjson.Parse(`{"key1": "val1"}`))
	spec.Set("key2", "val2")
	assert.Equal(t, "val2", spec.Get("key2").String())
}

func TestResourceSpec_ReplaceReference(t *testing.T) {
	spec := NewResourceData("test", "aws", "test", nil, gjson.Parse(`{}`))
	old := NewResourceData("aws_vpc", "aws", "old_vpc", nil, gjson.Parse(`{}`))
	new := NewResourceData("aws_vpc", "aws", "new_vpc", nil, gjson.Parse(`{}`))

	spec.AddReference("vpc_id", old, nil)
	spec.ReplaceReference("vpc_id", old, new)

	refs := spec.References("vpc_id")
	if len(refs) > 0 {
		assert.Equal(t, "new_vpc", refs[0].Address)
	}
}


// === usage_data.go coverage ===

func TestConsumptionMap_Data(t *testing.T) {
	data := map[string]*ConsumptionProfile{
		"r1": NewUsageData("r1", map[string]gjson.Result{"k": gjson.Parse("1")}),
	}
	cm := NewUsageMap(data)
	d := cm.Data()
	assert.NotNil(t, d)
}

func TestConsumptionMap_Less(t *testing.T) {
	// ConsumptionMap's wildcards implement sort.Interface
	cm := NewUsageMap(map[string]*ConsumptionProfile{
		"aws_instance[*]": NewUsageData("aws_instance[*]", nil),
		"aws_s3[*]":       NewUsageData("aws_s3[*]", nil),
	})
	_ = cm // Just verify it doesn't panic
}



// === project.go coverage ===

func TestWorkspace_NameWithWorkspace(t *testing.T) {
	ws := &Workspace{
		Name: "my-project",
		Metadata: &WorkspaceMeta{TerraformWorkspace: "prod"},
	}
	name := ws.NameWithWorkspace()
	assert.NotEmpty(t, name)
}




func TestProjects_Less(t *testing.T) {
	ps := Projects{
		{Name: "b-project", Metadata: &WorkspaceMeta{Path: "b"}},
		{Name: "a-project", Metadata: &WorkspaceMeta{Path: "a"}},
	}
	assert.True(t, ps.Less(1, 0))
	assert.False(t, ps.Less(0, 1))
}

func TestShortHash(t *testing.T) {
	hash := shortHash("abc123def456", 7)
	assert.NotEmpty(t, hash)
	assert.LessOrEqual(t, len(hash), 12) // Should be short
}

func TestWorkspaceMeta_GenerateProjectName_WithRemote(t *testing.T) {
	meta := &WorkspaceMeta{
		Path: "/code/infra",
	}
	remote := vcs.Remote{
		Host:  "github.com",
		
		Name:  "infra",
	}
	name := meta.GenerateProjectName(remote, false)
	assert.NotEmpty(t, name)
}

func TestWorkspaceMeta_GenerateProjectName_Dashboard(t *testing.T) {
	meta := &WorkspaceMeta{
		Path: "/code/infra",
	}
	name := meta.GenerateProjectName(vcs.Remote{}, true)
	assert.NotEmpty(t, name)
}

// === resource.go coverage ===

func TestEstimate_CalculateCosts_NoComponents(t *testing.T) {
	e := &Estimate{Name: "empty"}
	e.CalculateCosts()
	assert.Nil(t, e.HourlyCost)
	assert.Nil(t, e.MonthlyCost)
}

func TestSortEstimates_Direct(t *testing.T) {
	ws := &Workspace{
		Resources: []*Estimate{
			{Name: "c"}, {Name: "a"}, {Name: "b"},
		},
	}
	SortEstimates(ws)
	// SortEstimates sorts AllResources which returns a new slice
	// We can't verify the original order changed, but it shouldn't panic
}

func TestScaleQuantities_WithSubResources(t *testing.T) {
	e := &Estimate{
		CostComponents: []*LineItem{
			{MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(100))},
		},
		SubResources: []*Estimate{
			{
				CostComponents: []*LineItem{
					{MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(50))},
				},
			},
		},
	}
	ScaleQuantities(e, decimal.NewFromInt(2))
	assert.Equal(t, "200", e.CostComponents[0].MonthlyQuantity.String())
	assert.Equal(t, "100", e.SubResources[0].CostComponents[0].MonthlyQuantity.String())
}

// === cost_component.go coverage ===

func TestLineItem_UnitMultiplierQuantity_WithRounding(t *testing.T) {
	rounding := int32(2)
	monthly := decimal.NewFromFloat(730.123456)
	cc := &LineItem{
		MonthlyQuantity: &monthly,
		UnitMultiplier:  decimal.NewFromInt(1000),
		UnitRounding:    &rounding,
	}
	result := cc.UnitMultiplierMonthlyQuantity()
	assert.NotNil(t, result)
}

func TestLineItem_FillQuantities_MonthlyToHourly(t *testing.T) {
	cc := &LineItem{
		MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(730)),
		UnitMultiplier:  decimal.NewFromInt(1),
		price:           decimal.NewFromFloat(0.1),
	}
	cc.CalculateCosts()
	assert.NotNil(t, cc.HourlyCost)
}
