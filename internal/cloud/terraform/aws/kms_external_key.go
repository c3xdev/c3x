package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNewKMSExternalKeyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_kms_external_key",
		CoreRFunc: NewKMSExternalKey,
	}
}

func NewKMSExternalKey(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.KMSExternalKey{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
