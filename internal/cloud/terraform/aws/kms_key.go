package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNewKMSKeyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_kms_key",
		CoreRFunc: NewKMSKey,
	}
}

func NewKMSKey(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.KMSKey{
		Address:               d.Address,
		Region:                d.Get("region").String(),
		CustomerMasterKeySpec: d.Get("customer_master_key_spec").String(),
	}
	return r
}
