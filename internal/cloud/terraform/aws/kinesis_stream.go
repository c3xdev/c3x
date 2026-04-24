package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getKinesisStreamRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_kinesis_stream",
		CoreRFunc: newKinesisStream,
	}
}

func newKinesisStream(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	StreamMode := d.Get("stream_mode_details.0.stream_mode").String()
	ShardCount := d.Get("shard_count").Int()

	return &aws.KinesisStream{
		Address:    d.Address,
		Region:     region,
		StreamMode: StreamMode,
		ShardCount: ShardCount,
	}
}
