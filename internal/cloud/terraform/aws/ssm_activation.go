package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSSMActivationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ssm_activation",
		CoreRFunc: NewSSMActivation,
	}
}

func NewSSMActivation(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.SSMActivation{
		Address:           d.Address,
		Region:            d.Get("region").String(),
		RegistrationLimit: d.Get("registration_limit").Int(),
	}
	return r
}
