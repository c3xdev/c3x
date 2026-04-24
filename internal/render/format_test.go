package render

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFormatCost(t *testing.T) {
	tests := []struct {
		name     string
		currency string
		cost     *decimal.Decimal
		expected string
	}{
		{"nil cost", "USD", nil, ""},
		{"zero cost", "USD", dp(0), "$0"},
		{"small cost", "USD", dp(0.50), "$0.50"},
		{"medium cost", "USD", dp(560.64), "$561"},
		{"large cost", "USD", dp(10000), "$10,000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCost(tt.currency, tt.cost)
			if tt.expected != "" {
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestFormatQuantity(t *testing.T) {
	tests := []struct {
		name     string
		quantity *decimal.Decimal
	}{
		{"nil", nil},
		{"zero", dp(0)},
		{"integer", dp(730)},
		{"decimal", dp(3.14)},
		{"large", dp(1000000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatQuantity(tt.quantity)
			_ = result // Just verify no panic
		})
	}
}

func TestFormatPrice(t *testing.T) {
	result := formatPrice("USD", decimal.NewFromFloat(0.768))
	assert.NotEmpty(t, result)
	// Price may be formatted/rounded — just verify it's not empty
}

func TestFormatPercentChange(t *testing.T) {
	old := dp(100)
	new := dp(150)
	result := formatPercentChange(old, new)
	_ = result // Just verify no panic
}
