package engine

import (
	"sort"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestPolicies_Sort_ByCost(t *testing.T) {
	cost1 := decimal.NewFromInt(100)
	cost2 := decimal.NewFromInt(200)
	policies := Policies{
		{ID: "1", Address: "addr1", Cost: &cost1},
		{ID: "2", Address: "addr2", Cost: &cost2},
	}
	sort.Sort(policies)
	assert.Equal(t, "2", policies[0].ID) // higher cost first (descending)
	assert.Equal(t, "1", policies[1].ID)
}

func TestPolicies_Sort_NilCost(t *testing.T) {
	cost1 := decimal.NewFromInt(100)
	policies := Policies{
		{ID: "2", Address: "b-addr", Cost: nil},
		{ID: "1", Address: "a-addr", Cost: &cost1},
	}
	sort.Sort(policies)
	// Sort should not panic with nil costs
}

func TestPolicies_Swap(t *testing.T) {
	policies := Policies{{ID: "1"}, {ID: "2"}}
	policies.Swap(0, 1)
	assert.Equal(t, "2", policies[0].ID)
	assert.Equal(t, "1", policies[1].ID)
}

func TestBlankCoreResource_PopulateUsage(t *testing.T) {
	blank := BlankCoreResource{Type: "test"}
	ud := &ConsumptionProfile{Address: "test"}
	blank.PopulateUsage(ud) // no-op, should not panic
}

func TestEstimate_NoPriceResource(t *testing.T) {
	e := &Estimate{
		Name:    "aws_vpc.main",
		NoPrice: true,
	}
	e.CalculateCosts()
	assert.Nil(t, e.HourlyCost)
	assert.Nil(t, e.MonthlyCost)
}

func TestEstimate_SkippedResource(t *testing.T) {
	e := &Estimate{
		Name:        "aws_unsupported.x",
		IsSkipped:   true,
		SkipMessage: "Not supported",
	}
	assert.True(t, e.IsSkipped)
	assert.Equal(t, "Not supported", e.SkipMessage)
}

func TestEstimate_Tags(t *testing.T) {
	tags := map[string]string{"env": "prod", "team": "infra"}
	defaultTags := map[string]string{"org": "acme"}
	e := &Estimate{
		Name:        "test",
		Tags:        &tags,
		DefaultTags: &defaultTags,
	}
	assert.Equal(t, "prod", (*e.Tags)["env"])
	assert.Equal(t, "acme", (*e.DefaultTags)["org"])
}

func TestLineItem_FillQuantities_HourlyToMonthly(t *testing.T) {
	cc := &LineItem{
		HourlyQuantity: decimalPtrTest(decimal.NewFromFloat(1)),
		UnitMultiplier: decimal.NewFromInt(1),
		price:          decimal.NewFromFloat(0.1),
	}
	cc.CalculateCosts()
	assert.NotNil(t, cc.MonthlyCost) // Should calculate monthly from hourly
}

func TestWorkspace_PartialResources(t *testing.T) {
	ws := &Workspace{
		PartialResources: []*UnpricedEntry{
			{Type: "aws_instance", Address: "aws_instance.web"},
			{Type: "aws_s3_bucket", Address: "aws_s3_bucket.data"},
		},
	}
	all := ws.AllPartialResources()
	assert.Len(t, all, 2)
}

func TestResourceSpec_References_Advanced(t *testing.T) {
	spec := &ResourceSpec{
		Type:    "test",
		Address: "test.main",
		ReferencesMap: map[string][]*ResourceSpec{
			"vpc_id": {{Type: "aws_vpc", Address: "aws_vpc.main"}},
		},
	}
	refs := spec.References("vpc_id")
	assert.Len(t, refs, 1)
	assert.Equal(t, "aws_vpc.main", refs[0].Address)

	empty := spec.References("nonexistent")
	assert.Empty(t, empty)
}

func TestTagPropagation(t *testing.T) {
	tp := &TagPropagation{
		Attribute: "tags",
		To:        "aws_instance",
	}
	assert.Equal(t, "tags", tp.Attribute)
	assert.Equal(t, "aws_instance", tp.To)
}

func TestVendorMeta(t *testing.T) {
	vm := VendorMeta{
		Name:     "aws",
		Filename: "main.tf",
	}
	assert.Equal(t, "aws", vm.Name)
	assert.Equal(t, "main.tf", vm.Filename)
}

func TestConsumptionParam(t *testing.T) {
	cp := ConsumptionParam{
		Key:   "memory_size_gb",
		Value: "1.5",
	}
	assert.Equal(t, "memory_size_gb", cp.Key)
	assert.Equal(t, "1.5", cp.Value)
}
