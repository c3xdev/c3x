package pricing

import (
	"sync"
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
)

func newTestRateLookup() *RateLookup {
	return &RateLookup{
		resources:  make(map[string]*notFoundData),
		components: make(map[string]int),
		mux:        &sync.RWMutex{},
	}
}

func TestRateLookup_MissingPricesLen(t *testing.T) {
	rl := newTestRateLookup()
	assert.Equal(t, 0, rl.MissingPricesLen())

	rl.resources["aws_instance"] = &notFoundData{}
	assert.Equal(t, 1, rl.MissingPricesLen())
}

func TestRateLookup_MissingPricesComponents(t *testing.T) {
	rl := newTestRateLookup()
	assert.Empty(t, rl.MissingPricesComponents())

	rl.components["aws_instance compute"] = 1
	result := rl.MissingPricesComponents()
	assert.Len(t, result, 1)
}

type mockCatalogItem struct {
	engine.BlankCoreResource
}

func (m mockCatalogItem) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "test", ValueType: engine.Int64}}
}

func TestChunkPartialResources(t *testing.T) {
	resources := make([]*engine.UnpricedEntry, 10)
	for i := range resources {
		resources[i] = &engine.UnpricedEntry{
			CoreResource: mockCatalogItem{},
		}
	}
	chunks := chunkPartialResourcesWithUsage(resources, 3)
	assert.Len(t, chunks, 4)
	assert.Len(t, chunks[0], 3)
	assert.Len(t, chunks[3], 1)
}

func TestChunkPartialResources_Empty(t *testing.T) {
	chunks := chunkPartialResourcesWithUsage(nil, 5)
	assert.Empty(t, chunks)
}

func TestChunkPartialResources_NoUsageSchema(t *testing.T) {
	resources := make([]*engine.UnpricedEntry, 5)
	for i := range resources {
		resources[i] = &engine.UnpricedEntry{} // no CoreResource
	}
	chunks := chunkPartialResourcesWithUsage(resources, 3)
	assert.Empty(t, chunks) // all filtered out
}

func TestFlattenUsageKeys(t *testing.T) {
	items := []*engine.ConsumptionField{
		{Key: "monthly_requests", ValueType: engine.Int64},
		{Key: "storage_gb", ValueType: engine.Float64},
	}
	keys := flattenUsageKeys(items)
	assert.Contains(t, keys, "monthly_requests")
	assert.Contains(t, keys, "storage_gb")
}

func TestFlattenUsageKeys_Empty(t *testing.T) {
	keys := flattenUsageKeys(nil)
	assert.Empty(t, keys)
}
