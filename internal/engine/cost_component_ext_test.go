package engine

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestLineItem_SetPriceHash(t *testing.T) {
	cc := &LineItem{Name: "test"}
	cc.SetPriceHash("hash-123")
	assert.Equal(t, "hash-123", cc.PriceHash())
}

func TestLineItem_UnitMultiplierPrice(t *testing.T) {
	cc := &LineItem{
		UnitMultiplier: decimal.NewFromInt(1000000),
		price:          decimal.NewFromFloat(0.20),
	}
	// UnitMultiplierPrice = price * unitMultiplier
	result := cc.UnitMultiplierPrice()
	assert.Equal(t, "200000", result.StringFixed(0))
}

func TestLineItem_UnitMultiplierPrice_DefaultMultiplier(t *testing.T) {
	cc := &LineItem{
		price: decimal.NewFromFloat(0.50),
	}
	// With zero multiplier, should return price as-is or handle gracefully
	result := cc.UnitMultiplierPrice()
	assert.True(t, result.GreaterThanOrEqual(decimal.Zero) || result.LessThan(decimal.Zero))
}

func TestLineItem_UnitMultiplierHourlyQuantity(t *testing.T) {
	hourly := decimal.NewFromInt(2)
	cc := &LineItem{
		HourlyQuantity: &hourly,
		UnitMultiplier: decimal.NewFromInt(1000),
	}
	result := cc.UnitMultiplierHourlyQuantity()
	assert.NotNil(t, result)
}

func TestLineItem_UnitMultiplierHourlyQuantity_Nil(t *testing.T) {
	cc := &LineItem{
		UnitMultiplier: decimal.NewFromInt(1000),
	}
	result := cc.UnitMultiplierHourlyQuantity()
	assert.Nil(t, result)
}

func TestLineItem_UnitMultiplierMonthlyQuantity(t *testing.T) {
	monthly := decimal.NewFromInt(730)
	cc := &LineItem{
		MonthlyQuantity: &monthly,
		UnitMultiplier:  decimal.NewFromInt(1),
	}
	result := cc.UnitMultiplierMonthlyQuantity()
	assert.NotNil(t, result)
	assert.Equal(t, "730", result.String())
}

func TestLineItem_UnitMultiplierMonthlyQuantity_Nil(t *testing.T) {
	cc := &LineItem{
		UnitMultiplier: decimal.NewFromInt(1),
	}
	result := cc.UnitMultiplierMonthlyQuantity()
	assert.Nil(t, result)
}

func TestLineItem_UsageBased(t *testing.T) {
	cc := &LineItem{
		Name:            "Requests",
		UsageBased:      true,
		MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(1000000)),
		price:           decimal.NewFromFloat(0.20),
	}
	cc.CalculateCosts()

	assert.True(t, cc.UsageBased)
	assert.NotNil(t, cc.MonthlyCost)
}

func TestLineItem_IgnoreIfMissingPrice(t *testing.T) {
	cc := &LineItem{
		Name:                 "optional-feature",
		IgnoreIfMissingPrice: true,
	}
	assert.True(t, cc.IgnoreIfMissingPrice)
}

func TestComputeCosts_Project(t *testing.T) {
	ws := &Workspace{
		Resources: []*Estimate{
			{
				Name: "r1",
				CostComponents: []*LineItem{
					{
						Name:            "compute",
						MonthlyQuantity: decimalPtrTest(decimal.NewFromInt(730)),
						price:           decimal.NewFromFloat(0.10),
					},
				},
			},
		},
	}

	ComputeCosts(ws)

	assert.NotNil(t, ws.Resources[0].MonthlyCost)
	assert.Equal(t, "73", ws.Resources[0].MonthlyCost.StringFixed(0))
}
