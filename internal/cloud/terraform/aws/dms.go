package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDMSRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_dms_replication_instance",
		CoreRFunc: NewDMSReplicationInstance,
	}
}

func NewDMSReplicationInstance(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.DMSReplicationInstance{
		Address:                  d.Address,
		MultiAZ:                  d.Get("multi_az").Bool(),
		AllocatedStorageGB:       d.Get("allocated_storage").Int(),
		ReplicationInstanceClass: d.Get("replication_instance_class").String(),
		Region:                   d.Get("region").String(),
	}
	return r
}
