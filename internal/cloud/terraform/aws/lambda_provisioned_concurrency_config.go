package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLambdaProvisionedConcurrencyConfigRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_lambda_provisioned_concurrency_config",
		CoreRFunc: NewLambdaProvisionedConcurrencyConfig,
	}
}

func NewLambdaProvisionedConcurrencyConfig(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	name := d.Get("function_name").String()
	provisionedConcurrentExecutions := d.Get("provisioned_concurrent_executions").Int()

	r := &aws.LambdaProvisionedConcurrencyConfig{
		Address:                         d.Address,
		Region:                          region,
		Name:                            name,
		ProvisionedConcurrentExecutions: provisionedConcurrentExecutions,
	}

	return r
}
