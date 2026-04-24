package pricing

import (
	"sync"
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
)

func TestRateLookup_LogWarnings_NoMissing(t *testing.T) {
	rl := &RateLookup{
		resources:  make(map[string]*notFoundData),
		components: make(map[string]int),
		mux:        &sync.RWMutex{},
	}
	// Should not panic with no missing prices
	rl.LogWarnings()
}

func TestChunkPartialResources_ExactMultiple(t *testing.T) {
	mock := mockCatalogItem{}
	resources := make([]*engine.UnpricedEntry, 9)
	for i := range resources {
		resources[i] = &engine.UnpricedEntry{CoreResource: mock}
	}
	chunks := chunkPartialResourcesWithUsage(resources, 3)
	assert.Len(t, chunks, 3)
	for _, c := range chunks {
		assert.Len(t, c, 3)
	}
}
