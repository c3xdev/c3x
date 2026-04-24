package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getKinesisAnalyticsV2ApplicationSnapshotRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_kinesisanalyticsv2_application_snapshot",
		CoreRFunc: NewKinesisAnalyticsV2ApplicationSnapshot,
	}
}

func NewKinesisAnalyticsV2ApplicationSnapshot(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.KinesisAnalyticsV2ApplicationSnapshot{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
