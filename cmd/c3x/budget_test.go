package main

import (
	"fmt"
	"testing"

	"github.com/c3xdev/c3x/internal/render"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func newTestCmd(budget float64, budgetIncrease float64) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Float64("budget", budget, "")
	cmd.Flags().Float64("budget-increase", budgetIncrease, "")
	if budget > 0 {
		_ = cmd.Flags().Set("budget", fmt.Sprintf("%f", budget))
	}
	if budgetIncrease > 0 {
		_ = cmd.Flags().Set("budget-increase", fmt.Sprintf("%f", budgetIncrease))
	}
	return cmd
}

func decPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

func TestCheckBudget_UnderBudget(t *testing.T) {
	cmd := newTestCmd(1000, 0)
	r := render.Report{
		Currency:         "USD",
		TotalMonthlyCost: decPtr(decimal.NewFromFloat(500)),
	}
	err := checkBudget(cmd, r)
	assert.NoError(t, err)
}

func TestCheckBudget_OverBudget(t *testing.T) {
	cmd := newTestCmd(1000, 0)
	r := render.Report{
		Currency:         "USD",
		TotalMonthlyCost: decPtr(decimal.NewFromFloat(1500)),
	}
	err := checkBudget(cmd, r)
	assert.Error(t, err)
	assert.IsType(t, &BudgetExceededError{}, err)
}

func TestCheckBudget_NoBudgetSet(t *testing.T) {
	cmd := newTestCmd(0, 0)
	r := render.Report{
		Currency:         "USD",
		TotalMonthlyCost: decPtr(decimal.NewFromFloat(99999)),
	}
	err := checkBudget(cmd, r)
	assert.NoError(t, err)
}

func TestCheckBudget_ExactBudget(t *testing.T) {
	cmd := newTestCmd(1000, 0)
	r := render.Report{
		Currency:         "USD",
		TotalMonthlyCost: decPtr(decimal.NewFromFloat(1000)),
	}
	err := checkBudget(cmd, r)
	assert.NoError(t, err) // Equal is not over
}

func TestCheckBudget_NilCost(t *testing.T) {
	cmd := newTestCmd(1000, 0)
	r := render.Report{
		Currency:         "USD",
		TotalMonthlyCost: nil,
	}
	err := checkBudget(cmd, r)
	assert.NoError(t, err) // nil cost should not trigger budget
}

func TestCheckBudgetIncrease_OverLimit(t *testing.T) {
	cmd := newTestCmd(0, 10)
	past := decimal.NewFromFloat(1000)
	diff := decimal.NewFromFloat(200)
	r := render.Report{
		Currency:             "USD",
		PastTotalMonthlyCost: &past,
		DiffTotalMonthlyCost: &diff,
	}
	err := checkBudget(cmd, r)
	assert.Error(t, err)
	assert.IsType(t, &BudgetExceededError{}, err)
}

func TestCheckBudgetIncrease_UnderLimit(t *testing.T) {
	cmd := newTestCmd(0, 20)
	past := decimal.NewFromFloat(1000)
	diff := decimal.NewFromFloat(100)
	r := render.Report{
		Currency:             "USD",
		PastTotalMonthlyCost: &past,
		DiffTotalMonthlyCost: &diff,
	}
	err := checkBudget(cmd, r)
	assert.NoError(t, err)
}
