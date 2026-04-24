package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLBRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "aws_lb",
		ReferenceAttributes: []string{
			"subnet_mapping.#.allocation_id",
		},
		CoreRFunc: NewLB,
	}
}

func getALBRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_alb",
		CoreRFunc: NewLB,
	}
}

func NewLB(d *engine.ResourceSpec) engine.CatalogItem {
	loadBalancerType := d.Get("load_balancer_type").String()
	if loadBalancerType == "" {
		// set the default load balancer type as given https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lb
		// this is done as parsing the raw HCL will not pick up the default but return a blank string.
		loadBalancerType = "application"
	}

	r := &aws.LB{
		Address:          d.Address,
		Region:           d.Get("region").String(),
		LoadBalancerType: loadBalancerType,
	}
	return r
}
