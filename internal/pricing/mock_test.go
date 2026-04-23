package pricing

import (
	"context"
	"sync"
	"testing"

	"github.com/c3xdev/c3x/internal/apiclient"
	"github.com/c3xdev/c3x/internal/settings"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func newTestRunContext() *settings.Session {
	ctx, _ := settings.NewRunContextFromEnv(context.Background())
	if ctx == nil {
		ctx = settings.EmptyRunContext()
	}
	ctx.Config.Currency = "USD"
	return ctx
}

func TestNewPriceFetcher(t *testing.T) {
	ctx := newTestRunContext()
	fetcher := NewPriceFetcher(ctx, false)
	assert.NotNil(t, fetcher)
	assert.Equal(t, 0, fetcher.MissingPricesLen())
}

func TestPriceFetcher_EmptyProject(t *testing.T) {
	ctx := newTestRunContext()
	fetcher := NewPriceFetcher(ctx, false)

	project := &engine.Workspace{
		Name:     "empty",
		Metadata: &engine.WorkspaceMeta{Path: "/code"},
	}

	// PopulatePrices with no resources should not error
	err := fetcher.PopulatePrices(project)
	assert.NoError(t, err)
}

func TestPriceFetcher_SetCostComponentPrice_CustomPrice(t *testing.T) {
	ctx := newTestRunContext()
	fetcher := NewPriceFetcher(ctx, false)

	cc := &engine.LineItem{
		Name: "compute",
		ProductFilter: &engine.ProductSelector{},
	}
	resource := &engine.Estimate{Name: "test", ResourceType: "aws_instance"}

	customPrice := decimal.NewFromFloat(0.768)
	cc.SetCustomPrice(&customPrice)

	result := apiclientPriceQueryResult(resource, cc)
	fetcher.setCostComponentPrice(result)

	assert.Equal(t, "0.768", cc.Price().String())
}

func TestPriceFetcher_LogWarnings_Empty(t *testing.T) {
	ctx := newTestRunContext()
	fetcher := NewPriceFetcher(ctx, true)
	fetcher.LogWarnings() // Should not panic
}

func TestPriceFetcher_MissingPricesComponents_Empty(t *testing.T) {
	rl := &RateLookup{
		resources:  make(map[string]*notFoundData),
		components: make(map[string]int),
		mux:        &sync.RWMutex{},
	}
	assert.Empty(t, rl.MissingPricesComponents())
}

func TestPriceFetcher_WarnOnPriceErrors(t *testing.T) {
	ctx := newTestRunContext()

	// With warnOnPriceErrors = true
	fetcher1 := NewPriceFetcher(ctx, true)
	assert.NotNil(t, fetcher1)

	// With warnOnPriceErrors = false
	fetcher2 := NewPriceFetcher(ctx, false)
	assert.NotNil(t, fetcher2)
}

// Helper to create a PriceQueryResult
func apiclientPriceQueryResult(r *engine.Estimate, cc *engine.LineItem) apiclient.PriceQueryResult {
	return apiclient.PriceQueryResult{
		PriceQueryKey: apiclient.PriceQueryKey{
			Resource:      r,
			CostComponent: cc,
		},
	}
}
