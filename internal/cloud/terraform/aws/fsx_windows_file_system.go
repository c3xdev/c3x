package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getFSxWindowsFSRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_fsx_windows_file_system",
		Notes:     []string{"Data deduplication is not supported by Terraform."},
		CoreRFunc: NewFSxWindowsFileSystem,
	}
}
func NewFSxWindowsFileSystem(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.FSxWindowsFileSystem{
		Address:            d.Address,
		Region:             d.Get("region").String(),
		DeploymentType:     d.Get("deployment_type").String(),
		StorageType:        d.Get("storage_type").String(),
		ThroughputCapacity: d.Get("throughput_capacity").Int(),
		StorageCapacityGB:  d.Get("storage_capacity").Int(),
	}
	return r
}
