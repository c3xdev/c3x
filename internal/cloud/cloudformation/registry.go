package cloudformation

import (
	"sync"

	"github.com/c3xdev/c3x/internal/engine"

	"github.com/c3xdev/c3x/internal/cloud/cloudformation/aws"
)

type ResourceRegistryMap map[string]*engine.CatalogEntry

var (
	resourceRegistryMap ResourceRegistryMap
	once                sync.Once
)

func GetResourceRegistryMap() *ResourceRegistryMap {
	once.Do(func() {
		resourceRegistryMap = make(ResourceRegistryMap)

		// Merge all resource registries
		for _, registryItem := range aws.ResourceRegistry {
			resourceRegistryMap[registryItem.Name] = registryItem
		}
		for _, registryItem := range createFreeResources(aws.FreeResources) {
			resourceRegistryMap[registryItem.Name] = registryItem
		}
	})

	return &resourceRegistryMap
}

func GetUsageOnlyResources() []string {
	r := []string{}
	r = append(r, aws.UsageOnlyResources...)
	return r
}

func createFreeResources(l []string) []*engine.CatalogEntry {
	freeResources := make([]*engine.CatalogEntry, 0)
	for _, resourceName := range l {
		freeResources = append(freeResources, &engine.CatalogEntry{
			Name:    resourceName,
			NoPrice: true,
			Notes:   []string{"Free resource."},
		})
	}
	return freeResources
}
