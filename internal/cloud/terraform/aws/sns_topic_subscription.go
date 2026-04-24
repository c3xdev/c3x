package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSNSTopicSubscriptionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_sns_topic_subscription",
		CoreRFunc: NewSNSTopicSubscription,
		Notes: []string{
			"DEPRECATED.  Set subscription usage on aws_sns_topic instead.",
		},
		ReferenceAttributes: []string{"topic_arn"},
	}
}

func NewSNSTopicSubscription(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.SNSTopicSubscription{
		Address:  d.Address,
		Region:   d.Get("region").String(),
		Protocol: d.Get("protocol").String(),
	}
	return r
}
