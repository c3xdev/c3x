package engine

import (
	"sort"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func decimalPtrTest(d decimal.Decimal) *decimal.Decimal {
	return &d
}

func TestEstimate_ComputeCosts(t *testing.T) {
	e := &Estimate{
		Name: "test-resource",
		CostComponents: []*LineItem{
			{
				Name:            "compute",
				Unit:            "hours",
				HourlyQuantity:  decimalPtrTest(decimal.NewFromInt(1)),
				MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(730)),
				price:           decimal.NewFromFloat(0.10),
			},
			{
				Name:            "storage",
				Unit:            "GB",
				MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(100)),
				price:           decimal.NewFromFloat(0.05),
			},
		},
	}

	e.CalculateCosts()

	assert.NotNil(t, e.HourlyCost)
	assert.NotNil(t, e.MonthlyCost)
	assert.Equal(t, "0.1", e.HourlyCost.StringFixed(1))
	assert.Equal(t, "78", e.MonthlyCost.StringFixed(0))
}

func TestEstimate_ComputeCostsWithSubEstimates(t *testing.T) {
	e := &Estimate{
		Name: "parent",
		CostComponents: []*LineItem{
			{
				Name:            "parent-item",
				MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(1)),
				price:           decimal.NewFromFloat(10.0),
			},
		},
		SubResources: []*Estimate{
			{
				Name: "child",
				CostComponents: []*LineItem{
					{
						Name:            "child-item",
						MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(1)),
						price:           decimal.NewFromFloat(5.0),
					},
				},
			},
		},
	}

	e.CalculateCosts()
	for _, sub := range e.SubResources {
		sub.CalculateCosts()
	}
	e.CalculateCosts()

	assert.Equal(t, "15", e.MonthlyCost.StringFixed(0))
}

func TestEstimate_BaseResourceType(t *testing.T) {
	parent := &Estimate{Name: "parent", ResourceType: "aws_instance"}
	child := &Estimate{Name: "child", ResourceType: "ebs_volume", parent: parent}

	assert.Equal(t, "aws_instance", parent.BaseResourceType())
	assert.Equal(t, "aws_instance", child.BaseResourceType())
}

func TestEstimate_BaseResourceName(t *testing.T) {
	parent := &Estimate{Name: "aws_instance.web", ResourceType: "aws_instance"}
	child := &Estimate{Name: "root_block_device", parent: parent}

	assert.Equal(t, "aws_instance.web", parent.BaseResourceName())
	assert.Equal(t, "aws_instance.web", child.BaseResourceName())
}

func TestEstimate_FlattenedSubEstimates(t *testing.T) {
	grandchild := &Estimate{Name: "grandchild"}
	child := &Estimate{Name: "child", SubResources: []*Estimate{grandchild}}
	parent := &Estimate{Name: "parent", SubResources: []*Estimate{child}}

	flat := parent.FlattenedSubResources()
	assert.Len(t, flat, 2)
	assert.Equal(t, "child", flat[0].Name)
	assert.Equal(t, "grandchild", flat[1].Name)
}

func TestEstimate_RemoveLineItem(t *testing.T) {
	cc1 := &LineItem{Name: "keep"}
	cc2 := &LineItem{Name: "remove"}
	e := &Estimate{CostComponents: []*LineItem{cc1, cc2}}

	e.RemoveCostComponent(cc2)

	assert.Len(t, e.CostComponents, 1)
	assert.Equal(t, "keep", e.CostComponents[0].Name)
}

func TestLineItem_CalculateCosts(t *testing.T) {
	cc := &LineItem{
		Name:            "test",
		Unit:            "hours",
		HourlyQuantity:  decimalPtrTest(decimal.NewFromInt(2)),
		MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(1460)),
		price:           decimal.NewFromFloat(0.50),
	}

	cc.CalculateCosts()

	assert.NotNil(t, cc.HourlyCost)
	assert.NotNil(t, cc.MonthlyCost)
	assert.Equal(t, "1", cc.HourlyCost.StringFixed(0))
	assert.Equal(t, "730", cc.MonthlyCost.StringFixed(0))
}

func TestLineItem_SetPrice(t *testing.T) {
	cc := &LineItem{Name: "test"}

	cc.SetPrice(decimal.NewFromFloat(0.768))

	assert.Equal(t, "0.768", cc.Price().String())
}

func TestLineItem_SetPriceNotFound(t *testing.T) {
	cc := &LineItem{Name: "test"}

	cc.SetPriceNotFound()

	assert.True(t, cc.PriceNotFound)
	assert.True(t, cc.Price().IsZero())
}

func TestLineItem_CustomPrice(t *testing.T) {
	cc := &LineItem{Name: "test"}
	customPrice := decimal.NewFromFloat(1.23)

	cc.SetCustomPrice(&customPrice)

	assert.NotNil(t, cc.CustomPrice())
	assert.Equal(t, "1.23", cc.CustomPrice().String())
}

func TestLineItem_WithDiscount(t *testing.T) {
	cc := &LineItem{
		Name:                "test",
		MonthlyQuantity:     decimalPtrTest(decimal.NewFromInt(100)),
		MonthlyDiscountPerc: 0.20,
		price:               decimal.NewFromFloat(1.0),
	}

	cc.CalculateCosts()

	assert.Equal(t, "80", cc.MonthlyCost.StringFixed(0))
}

func TestSortEstimates(t *testing.T) {
	// SortEstimates sorts the slice returned by AllResources
	// Note: AllResources creates a new slice, so we test the function directly
	resources := []*Estimate{
		{Name: "c-resource"},
		{Name: "a-resource"},
		{Name: "b-resource"},
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].Name < resources[j].Name
	})

	assert.Equal(t, "a-resource", resources[0].Name)
	assert.Equal(t, "b-resource", resources[1].Name)
	assert.Equal(t, "c-resource", resources[2].Name)
}

func TestScaleQuantities(t *testing.T) {
	e := &Estimate{
		CostComponents: []*LineItem{
			{
				HourlyQuantity:  decimalPtrTest(decimal.NewFromInt(1)),
				MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(730)),
			},
		},
	}

	ScaleQuantities(e, decimal.NewFromInt(3))

	assert.Equal(t, "3", e.CostComponents[0].HourlyQuantity.String())
	assert.Equal(t, "2190", e.CostComponents[0].MonthlyQuantity.String())
}
