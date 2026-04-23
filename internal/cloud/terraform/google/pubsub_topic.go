package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPubSubTopicRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_pubsub_topic",
		CoreRFunc: NewPubSubTopic,
	}
}

func NewPubSubTopic(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.PubSubTopic{
		Address: d.Address,
	}

	return r
}
