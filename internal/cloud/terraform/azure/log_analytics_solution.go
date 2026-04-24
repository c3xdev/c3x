package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getLogAnalyticsSolutionRegistryItem() *engine.CatalogEntry {
	refs := []string{
		"resource_group_name",
		"workspace_resource_id",
	}

	return &engine.CatalogEntry{
		Name:                "azurerm_log_analytics_solution",
		RFunc:               newLogAnalyticsSolution,
		ReferenceAttributes: append(refs, sentinelDataConnectorRefs...),
	}
}

func newLogAnalyticsSolution(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	return &engine.Estimate{
		Name:        d.Address,
		IsSkipped:   true,
		NoPrice:     true,
		UsageSchema: []*engine.ConsumptionField{},
	}
}
