package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getRedisInstanceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_redis_instance",
		CoreRFunc: NewRedisInstance,
	}
}

func NewRedisInstance(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.RedisInstance{
		Address:      d.Address,
		Region:       d.Get("region").String(),
		MemorySizeGB: d.Get("memory_size_gb").Float(),
		Tier:         d.Get("tier").String(),
	}

	return r
}
