package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEFSFileSystemRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_efs_file_system",
		CoreRFunc: NewEFSFileSystem,
	}
}
func NewEFSFileSystem(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.EFSFileSystem{
		Address:                     d.Address,
		Region:                      d.Get("region").String(),
		HasLifecyclePolicy:          len(d.Get("lifecycle_policy").Array()) > 0,
		AvailabilityZoneName:        d.Get("availability_zone_name").String(),
		ProvisionedThroughputInMBps: d.Get("provisioned_throughput_in_mibps").Float(),
	}
	return r
}
