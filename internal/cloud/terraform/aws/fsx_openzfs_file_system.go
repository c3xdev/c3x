package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getFSxOpenZFSFSRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_fsx_openzfs_file_system",
		Notes:     []string{"Data deduplication is not supported by Terraform."},
		CoreRFunc: NewFSxOpenZFSFileSystem,
	}
}
func NewFSxOpenZFSFileSystem(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.FSxOpenZFSFileSystem{
		Address:             d.Address,
		Region:              d.Get("region").String(),
		DeploymentType:      d.Get("deployment_type").String(),
		StorageType:         d.Get("storage_type").String(),
		ThroughputCapacity:  d.Get("throughput_capacity").Int(),
		StorageCapacityGB:   d.Get("storage_capacity").Int(),
		ProvisionedIOPS:     d.Get("disk_iops_configuration.0.iops").Int(),
		ProvisionedIOPSMode: d.Get("disk_iops_configuration.0.mode").String(),
		DataCompression:     d.Get("root_volume_configuration.0.data_compression_type").String(),
	}
	return r
}
