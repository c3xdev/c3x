package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSecretsManagerSecret() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_secretsmanager_secret",
		CoreRFunc: NewSecretsManagerSecret,
	}
}

func NewSecretsManagerSecret(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.SecretsManagerSecret{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
