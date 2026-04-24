package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudHSMv2HSMRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_cloudhsm_v2_hsm",
		CoreRFunc: newCloudHSMv2HSM,
	}
}

func newCloudHSMv2HSM(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	return &aws.CloudHSMv2HSM{
		Address: d.Address,
		Region:  region,
	}
}
