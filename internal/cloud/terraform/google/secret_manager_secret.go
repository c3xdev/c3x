package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/tidwall/gjson"
)

func getSecretManagerSecretRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_secret_manager_secret",
		CoreRFunc: newSecretManagerSecret,
	}
}

func newSecretManagerSecret(d *engine.ResourceSpec) engine.CatalogItem {
	return &google.SecretManagerSecret{
		Address:              d.Address,
		Region:               d.Get("region").String(),
		ReplicationLocations: secretManagerSecretReplicasCount(d),
	}
}

func secretManagerSecretReplicasCount(d *engine.ResourceSpec) int64 {
	replicasCount := 1

	replications := d.Get("replication.0.user_managed.0")
	if replications.Type != gjson.Null && len(replications.Array()) > 0 {
		replicasCount = len(replications.Get("replicas").Array())
	}

	return int64(replicasCount)
}
