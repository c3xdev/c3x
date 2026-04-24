package engine

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestObservedCost_Fields(t *testing.T) {
	now := time.Now()
	cost := ObservedCost{
		ResourceID:     "i-12345",
		StartTimestamp: now,
		EndTimestamp:   now.Add(24 * time.Hour),
		CostComponents: []*LineItem{
			{
				Name:            "compute",
				MonthlyCost:     decimalPtrTest(decimal.NewFromFloat(560.64)),
				MonthlyQuantity: decimalPtrTest(decimal.NewFromFloat(730)),
				price:           decimal.NewFromFloat(0.768),
				Unit:            "hours",
			},
		},
	}

	assert.Equal(t, "i-12345", cost.ResourceID)
	assert.False(t, cost.StartTimestamp.IsZero())
	assert.Len(t, cost.CostComponents, 1)
	assert.Equal(t, "compute", cost.CostComponents[0].Name)
}
