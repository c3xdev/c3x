package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSNSTopicRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_sns_topic",
		CoreRFunc:           NewSNSTopic,
		ReferenceAttributes: []string{"aws_sns_topic_subscription.topic_arn"},
	}
}

func NewSNSTopic(d *engine.ResourceSpec) engine.CatalogItem {
	if d.GetBoolOrDefault("fifo_topic", false) {
		r := &aws.SNSFIFOTopic{
			Address:       d.Address,
			Region:        d.Get("region").String(),
			Subscriptions: int64(len(d.References("aws_sns_topic_subscription.topic_arn"))),
		}

		return r
	}

	r := &aws.SNSTopic{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
