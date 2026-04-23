package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPubSubSubscriptionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_pubsub_subscription",
		CoreRFunc: NewPubSubSubscription,
	}
}

func NewPubSubSubscription(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.PubSubSubscription{
		Address: d.Address,
	}

	return r
}
