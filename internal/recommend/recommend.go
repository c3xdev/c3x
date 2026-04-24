// Package recommend provides cost optimization recommendations based on estimated resources.
package recommend

import (
	"fmt"
	"strings"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

// Recommendation represents a single cost optimization suggestion.
type Recommendation struct {
	ResourceName   string           `json:"resourceName"`
	ResourceType   string           `json:"resourceType"`
	Category       string           `json:"category"`
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	CurrentCost    *decimal.Decimal `json:"currentMonthlyCost"`
	SuggestedCost  *decimal.Decimal `json:"suggestedMonthlyCost,omitempty"`
	MonthlySavings *decimal.Decimal `json:"monthlySavings,omitempty"`
	SavingsPercent *decimal.Decimal `json:"savingsPercent,omitempty"`
}

// Result holds all recommendations for a set of projects.
type Result struct {
	Recommendations  []Recommendation `json:"recommendations"`
	TotalMonthlyCost *decimal.Decimal `json:"totalMonthlyCost"`
	PotentialSavings *decimal.Decimal `json:"potentialMonthlySavings"`
	SavingsPercent   *decimal.Decimal `json:"savingsPercent"`
}

// Analyze examines projects and returns cost optimization recommendations.
func Analyze(projects []*engine.Workspace) *Result {
	result := &Result{
		Recommendations: make([]Recommendation, 0),
	}

	totalCost := decimal.Zero
	totalSavings := decimal.Zero

	for _, project := range projects {
		for _, resource := range project.Resources {
			if resource.MonthlyCost != nil {
				totalCost = totalCost.Add(*resource.MonthlyCost)
			}

			recs := analyzeResource(resource)
			result.Recommendations = append(result.Recommendations, recs...)

			for _, rec := range recs {
				if rec.MonthlySavings != nil {
					totalSavings = totalSavings.Add(*rec.MonthlySavings)
				}
			}
		}
	}

	result.TotalMonthlyCost = &totalCost
	result.PotentialSavings = &totalSavings
	if totalCost.IsPositive() {
		pct := totalSavings.Div(totalCost).Mul(decimal.NewFromInt(100))
		result.SavingsPercent = &pct
	}

	return result
}

func analyzeResource(resource *engine.Estimate) []Recommendation {
	var recs []Recommendation

	switch {
	case strings.HasPrefix(resource.ResourceType, "aws_"):
		recs = append(recs, analyzeAWS(resource)...)
	case strings.HasPrefix(resource.ResourceType, "azurerm_"):
		recs = append(recs, analyzeAzure(resource)...)
	case strings.HasPrefix(resource.ResourceType, "google_"):
		recs = append(recs, analyzeGoogle(resource)...)
	}

	return recs
}

func decPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}

// FormatTable returns a human-readable table of recommendations.
func FormatTable(result *Result, noColor bool) string {
	if len(result.Recommendations) == 0 {
		return "No optimization recommendations found.\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%d recommendation(s) found:\n\n", len(result.Recommendations)))

	for i, rec := range result.Recommendations {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, rec.Title))
		sb.WriteString(fmt.Sprintf("     Resource: %s (%s)\n", rec.ResourceName, rec.ResourceType))
		sb.WriteString(fmt.Sprintf("     %s\n", rec.Description))
		if rec.MonthlySavings != nil && rec.SavingsPercent != nil {
			sb.WriteString(fmt.Sprintf("     Potential savings: $%s/mo (%s%%)\n",
				rec.MonthlySavings.StringFixed(2),
				rec.SavingsPercent.StringFixed(0)))
		}
		sb.WriteString("\n")
	}

	if result.PotentialSavings != nil && result.PotentialSavings.IsPositive() && result.TotalMonthlyCost != nil {
		sb.WriteString(fmt.Sprintf("  Total potential savings: $%s/mo", result.PotentialSavings.StringFixed(2)))
		if result.SavingsPercent != nil {
			sb.WriteString(fmt.Sprintf(" (%s%% of $%s/mo estimated cost)",
				result.SavingsPercent.StringFixed(0),
				result.TotalMonthlyCost.StringFixed(2)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
