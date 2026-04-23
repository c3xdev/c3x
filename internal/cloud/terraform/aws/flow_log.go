package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getFlowLogRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "aws_flow_log",
		CoreRFunc: func(d *engine.ResourceSpec) engine.CatalogItem {
			return engine.BlankCoreResource{
				Name: d.Address,
				Type: d.Type,
			}
		},
	}
}
