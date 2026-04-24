package main

import (
	"fmt"

	"github.com/c3xdev/c3x/internal/render"
	"github.com/c3xdev/c3x/internal/ui"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

// BudgetExceededError is returned when a cost estimate exceeds the configured budget.
// It causes the CLI to exit with code 1, making it useful as a CI/CD gate.
type BudgetExceededError struct {
	Message string
}

func (e *BudgetExceededError) Error() string {
	return e.Message
}

func checkBudget(cmd *cobra.Command, r render.Report) error {
	budget, _ := cmd.Flags().GetFloat64("budget")
	budgetIncrease, _ := cmd.Flags().GetFloat64("budget-increase")

	if budget > 0 && r.TotalMonthlyCost != nil {
		budgetDec := decimal.NewFromFloat(budget)
		if r.TotalMonthlyCost.GreaterThan(budgetDec) {
			return &BudgetExceededError{
				Message: fmt.Sprintf(
					"%s Monthly cost estimate %s exceeds budget of %s %s",
					ui.WarningString("Budget exceeded:"),
					ui.PrimaryString(render.FormatCost2DP(r.Currency, r.TotalMonthlyCost)),
					ui.PrimaryString(render.FormatCost2DP(r.Currency, &budgetDec)),
					"(use --budget to adjust)",
				),
			}
		}
	}

	if budgetIncrease > 0 && r.DiffTotalMonthlyCost != nil && r.PastTotalMonthlyCost != nil {
		if r.PastTotalMonthlyCost.IsPositive() {
			pctChange := r.DiffTotalMonthlyCost.Div(*r.PastTotalMonthlyCost).Mul(decimal.NewFromInt(100))
			limitDec := decimal.NewFromFloat(budgetIncrease)
			if pctChange.GreaterThan(limitDec) {
				return &BudgetExceededError{
					Message: fmt.Sprintf(
						"%s Cost increase of %s%% exceeds limit of %s%% %s",
						ui.WarningString("Budget increase exceeded:"),
						ui.PrimaryString(pctChange.StringFixed(1)),
						ui.PrimaryString(limitDec.StringFixed(1)),
						"(use --budget-increase to adjust)",
					),
				}
			}
		}
	}

	return nil
}
