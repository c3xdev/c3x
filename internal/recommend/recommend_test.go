package recommend

import (
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestAnalyze_Empty(t *testing.T) {
	result := Analyze([]*engine.Workspace{})
	assert.Empty(t, result.Recommendations)
	assert.True(t, result.TotalMonthlyCost.IsZero())
}

func TestAnalyze_NoRecommendations(t *testing.T) {
	project := &engine.Workspace{
		Resources: []*engine.Estimate{
			{
				Name:         "aws_instance.web",
				ResourceType: "aws_instance",
				MonthlyCost:  decPtr(decimal.NewFromFloat(100)),
				CostComponents: []*engine.LineItem{
					{Name: "Instance usage (Linux/UNIX, on-demand, m7i.xlarge)"},
				},
			},
		},
	}
	result := Analyze([]*engine.Workspace{project})
	assert.Empty(t, result.Recommendations)
}

func TestAnalyze_AWSInstanceGeneration(t *testing.T) {
	project := &engine.Workspace{
		Resources: []*engine.Estimate{
			{
				Name:         "aws_instance.web",
				ResourceType: "aws_instance",
				MonthlyCost:  decPtr(decimal.NewFromFloat(560)),
				CostComponents: []*engine.LineItem{
					{Name: "Instance usage (Linux/UNIX, on-demand, m5.xlarge)"},
				},
			},
		},
	}
	result := Analyze([]*engine.Workspace{project})
	assert.Len(t, result.Recommendations, 1)
	assert.Equal(t, "instance-generation", result.Recommendations[0].Category)
	assert.Contains(t, result.Recommendations[0].Title, "m5.xlarge")
	assert.Contains(t, result.Recommendations[0].Title, "m7i.xlarge")
}

func TestAnalyze_AWSEBSgp2(t *testing.T) {
	project := &engine.Workspace{
		Resources: []*engine.Estimate{
			{
				Name:         "aws_ebs_volume.data",
				ResourceType: "aws_ebs_volume",
				MonthlyCost:  decPtr(decimal.NewFromFloat(50)),
				CostComponents: []*engine.LineItem{
					{Name: "Storage (general purpose SSD, gp2)"},
				},
			},
		},
	}
	result := Analyze([]*engine.Workspace{project})
	assert.Len(t, result.Recommendations, 1)
	assert.Equal(t, "storage-type", result.Recommendations[0].Category)
	assert.Contains(t, result.Recommendations[0].Title, "gp3")
}

func TestAnalyze_AWSNATGateway(t *testing.T) {
	project := &engine.Workspace{
		Resources: []*engine.Estimate{
			{
				Name:         "aws_nat_gateway.main",
				ResourceType: "aws_nat_gateway",
				MonthlyCost:  decPtr(decimal.NewFromFloat(32)),
				CostComponents: []*engine.LineItem{
					{Name: "NAT gateway"},
				},
			},
		},
	}
	result := Analyze([]*engine.Workspace{project})
	assert.Len(t, result.Recommendations, 1)
	assert.Equal(t, "architecture", result.Recommendations[0].Category)
	assert.Contains(t, result.Recommendations[0].Title, "VPC endpoints")
}

func TestAnalyze_AWSRDSGeneration(t *testing.T) {
	project := &engine.Workspace{
		Resources: []*engine.Estimate{
			{
				Name:         "aws_db_instance.main",
				ResourceType: "aws_db_instance",
				MonthlyCost:  decPtr(decimal.NewFromFloat(200)),
				CostComponents: []*engine.LineItem{
					{Name: "Database instance (on-demand, db.m5.large)"},
				},
			},
		},
	}
	result := Analyze([]*engine.Workspace{project})
	assert.Len(t, result.Recommendations, 1)
	assert.Contains(t, result.Recommendations[0].Title, "db.m7i.large")
}

func TestAnalyze_AzureVMGeneration(t *testing.T) {
	project := &engine.Workspace{
		Resources: []*engine.Estimate{
			{
				Name:         "azurerm_linux_virtual_machine.web",
				ResourceType: "azurerm_linux_virtual_machine",
				MonthlyCost:  decPtr(decimal.NewFromFloat(150)),
				CostComponents: []*engine.LineItem{
					{Name: "Instance usage (pay as you go, Standard_D4_v3)"},
				},
			},
		},
	}
	result := Analyze([]*engine.Workspace{project})
	assert.Len(t, result.Recommendations, 1)
	assert.Contains(t, result.Recommendations[0].Title, "Standard_D4_v5")
}

func TestAnalyze_GoogleInstanceGeneration(t *testing.T) {
	project := &engine.Workspace{
		Resources: []*engine.Estimate{
			{
				Name:         "google_compute_instance.web",
				ResourceType: "google_compute_instance",
				MonthlyCost:  decPtr(decimal.NewFromFloat(100)),
				CostComponents: []*engine.LineItem{
					{Name: "Instance usage (on-demand, n1-standard-4)"},
				},
			},
		},
	}
	result := Analyze([]*engine.Workspace{project})
	assert.Len(t, result.Recommendations, 1)
	assert.Contains(t, result.Recommendations[0].Title, "n2-standard-4")
}

func TestAnalyze_MultipleResources(t *testing.T) {
	project := &engine.Workspace{
		Resources: []*engine.Estimate{
			{
				Name:         "aws_instance.web",
				ResourceType: "aws_instance",
				MonthlyCost:  decPtr(decimal.NewFromFloat(560)),
				CostComponents: []*engine.LineItem{
					{Name: "Instance usage (Linux/UNIX, on-demand, m5.xlarge)"},
				},
			},
			{
				Name:         "aws_ebs_volume.data",
				ResourceType: "aws_ebs_volume",
				MonthlyCost:  decPtr(decimal.NewFromFloat(50)),
				CostComponents: []*engine.LineItem{
					{Name: "Storage (general purpose SSD, gp2)"},
				},
			},
			{
				Name:         "aws_nat_gateway.main",
				ResourceType: "aws_nat_gateway",
				MonthlyCost:  decPtr(decimal.NewFromFloat(32)),
				CostComponents: []*engine.LineItem{
					{Name: "NAT gateway"},
				},
			},
		},
	}
	result := Analyze([]*engine.Workspace{project})
	assert.Len(t, result.Recommendations, 3)
	assert.True(t, result.TotalMonthlyCost.Equal(decimal.NewFromFloat(642)))
}

func TestFormatTable_Empty(t *testing.T) {
	result := &Result{Recommendations: []Recommendation{}}
	output := FormatTable(result, true)
	assert.Contains(t, output, "No optimization recommendations found")
}

func TestFormatTable_WithRecommendations(t *testing.T) {
	result := &Result{
		Recommendations: []Recommendation{
			{
				ResourceName: "aws_instance.web",
				ResourceType: "aws_instance",
				Category:     "instance-generation",
				Title:        "Upgrade to m7i.xlarge",
				Description:  "Better price-performance.",
			},
		},
		TotalMonthlyCost: decPtr(decimal.NewFromFloat(100)),
		PotentialSavings: decPtr(decimal.NewFromFloat(10)),
		SavingsPercent:   decPtr(decimal.NewFromFloat(10)),
	}
	output := FormatTable(result, true)
	assert.Contains(t, output, "1 recommendation(s) found")
	assert.Contains(t, output, "Upgrade to m7i.xlarge")
	assert.Contains(t, output, "Total potential savings")
}
