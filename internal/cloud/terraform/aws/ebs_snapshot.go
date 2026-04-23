package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEBSSnapshotRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_ebs_snapshot",
		CoreRFunc:           NewEBSSnapshot,
		ReferenceAttributes: []string{"volume_id"},
	}
}
func NewEBSSnapshot(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.EBSSnapshot{Address: d.Address, Region: d.Get("region").String()}
	volumeRefs := d.References("volume_id")
	if len(volumeRefs) > 0 {
		if volumeRefs[0].Get("size").Exists() {
			r.SizeGB = floatPtr(volumeRefs[0].Get("size").Float())
		}
	}
	return r
}
