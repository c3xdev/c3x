package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getOpensearchDomainRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_opensearch_domain",
		CoreRFunc: newSearchDomain,
	}
}
