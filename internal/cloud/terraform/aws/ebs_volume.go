package aws

import (
	"github.com/tidwall/gjson"

	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEBSVolumeRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ebs_volume",
		CoreRFunc: NewEBSVolume,
	}
}

func NewEBSVolume(d *engine.ResourceSpec) engine.CatalogItem {
	var size *int64
	if d.Get("size").Type != gjson.Null {
		size = intPtr(d.Get("size").Int())
	}

	a := &aws.EBSVolume{
		Address:    d.Address,
		Region:     d.Get("region").String(),
		Type:       d.Get("type").String(),
		IOPS:       d.Get("iops").Int(),
		Throughput: d.Get("throughput").Int(),
		Size:       size,
	}

	return a
}
