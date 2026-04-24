package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSQSQueueRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_sqs_queue",
		CoreRFunc: NewSQSQueue,
	}
}

func NewSQSQueue(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.SQSQueue{
		Address:   d.Address,
		Region:    d.Get("region").String(),
		FifoQueue: d.Get("fifo_queue").Bool(),
	}
	return r
}
