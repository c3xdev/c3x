package aws

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getWAFWebACLRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_waf_web_acl",
		CoreRFunc: NewWAFWebACL,
		Notes: []string{
			"Seller fees for Managed Rule Groups from AWS Marketplace are not included. Bot Control is not supported by Terraform.",
		},
	}
}

func NewWAFWebACL(d *engine.ResourceSpec) engine.CatalogItem {
	rules := int64(0)
	ruleGroups := int64(0)

	for _, rule := range d.Get("rules").Array() {
		ruleType := rule.Get("type").String()

		if strings.ToLower(ruleType) == "regular" || strings.ToLower(ruleType) == "rate_based" {
			rules++
		} else if strings.ToLower(ruleType) == "group" {
			ruleGroups++
		}
	}

	r := &aws.WAFWebACL{
		Address:    d.Address,
		Region:     d.Get("region").String(),
		Rules:      rules,
		RuleGroups: ruleGroups,
	}
	return r
}
