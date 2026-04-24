package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getConfigRuleItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_config_config_rule",
		CoreRFunc: NewConfigConfigRule,
	}
}
func NewConfigConfigRule(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.ConfigConfigRule{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
