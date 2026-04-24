package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEBSSnapshotCopyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ebs_snapshot_copy",
		CoreRFunc: NewEBSSnapshotCopy,
		ReferenceAttributes: []string{
			"volume_id",
			"source_snapshot_id",
		},
	}
}
func NewEBSSnapshotCopy(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.EBSSnapshotCopy{Address: d.Address, Region: d.Get("region").String()}
	sourceSnapshotRefs := d.References("source_snapshot_id")
	if len(sourceSnapshotRefs) > 0 {
		volumeRefs := sourceSnapshotRefs[0].References("volume_id")
		if len(volumeRefs) > 0 {
			if volumeRefs[0].Get("size").Exists() {
				r.SizeGB = floatPtr(volumeRefs[0].Get("size").Float())
			}
		}
	}
	return r
}
