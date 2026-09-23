// Package calculator orchestrates the cost-estimation pipeline. It owns
// no data; instead it wires together:
//
//   - a catalog Registry      (resource definitions)
//   - an expr evaluator        (quantity / rate / when expressions)
//   - a pricing Source         (per-unit rate lookups)
//
// The engine produces a domain.Estimate without knowing whether the
// resources came from Terraform, plan JSON, or CloudFormation —
// that's the parser's concern.
package calculator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/expr"
	"github.com/c3xdev/c3x/internal/observability"
	"github.com/c3xdev/c3x/internal/pricing"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
)

// Engine is the cost-estimation orchestrator. Construct via New so the
// catalog and pricing source are wired correctly.
type Engine struct {
	registry      *catalog.Registry
	prices        pricing.Source
	currency      domain.Currency
	defaultRegion string
	logger        *slog.Logger
	now           func() time.Time

	// programs caches compiled expressions across resources of the same
	// kind. Each catalog file's expressions are compiled exactly once
	// per Engine lifetime.
	programs *programCache
}

// Options configures a new Engine. All fields are optional; sensible
// defaults are applied for anything left zero-valued.
type Options struct {
	Registry      *catalog.Registry
	Prices        pricing.Source
	Currency      domain.Currency
	DefaultRegion string
	Logger        *slog.Logger
	// Now is the clock used for Estimate.GeneratedAt. Override in tests
	// to get reproducible timestamps.
	Now func() time.Time
}

// New constructs an Engine from Options. A nil Registry or Prices is a
// programming error; the function panics so the misuse is caught in
// development rather than producing a zero-cost estimate at runtime.
func New(opts Options) *Engine {
	if opts.Registry == nil {
		panic("calculator.New: Registry is required")
	}
	if opts.Prices == nil {
		panic("calculator.New: Prices is required")
	}
	if opts.Currency == domain.CurrencyUnknown {
		opts.Currency = domain.CurrencyUSD
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Engine{
		registry:      opts.Registry,
		prices:        opts.Prices,
		currency:      opts.Currency,
		defaultRegion: opts.DefaultRegion,
		logger:        opts.Logger,
		now:           opts.Now,
		programs:      newProgramCache(),
	}
}

// Estimate computes one Estimate over the given resources.
//
// Unknown resource kinds are not an error: they appear in the output as
// a zero-cost Cost with no LineItems. This lets the engine produce a
// useful breakdown even when the catalog doesn't price every resource
// the IaC declared (which is the steady state for any cost tool).
func (e *Engine) Estimate(ctx context.Context, resources []domain.Resource) (domain.Estimate, error) {
	ctx, span := observability.Tracer().Start(ctx, "calculator.Estimate")
	defer span.End()
	span.SetAttributes(attribute.Int("c3x.resource_count", len(resources)))

	costs := make([]domain.Cost, 0, len(resources))
	var skipped []domain.SkippedResource
	for _, r := range resources {
		c, reason, err := e.costFor(ctx, r)
		if err != nil {
			return domain.Estimate{}, fmt.Errorf("%s: %w", r.Ref.Label(), err)
		}
		costs = append(costs, c)
		// A non-empty reason means the engine produced a $0 cost
		// not because the resource is genuinely free but because
		// the catalog couldn't price it. Record so renderers can
		// surface the gap.
		if reason != "" {
			skipped = append(skipped, domain.SkippedResource{
				Resource: r.Ref, Reason: reason,
			})
		}
	}
	est := domain.NewEstimate(costs, e.currency, e.now())
	est.Skipped = skipped
	span.SetAttributes(attribute.String("c3x.project_total", est.ProjectTotal.String()))
	return est, nil
}

// costFor builds the Cost for one resource by iterating its catalog
// dimensions. Each dimension contributes zero or one LineItem.
//
// The second return value is a non-empty "skip reason" when the
// resource produced no priced output for a fixable cause — the
// caller (Estimate) collects these so renderers can surface them
// via --show-skipped.
func (e *Engine) costFor(ctx context.Context, r domain.Resource) (domain.Cost, string, error) {
	ctx, span := observability.Tracer().Start(ctx, "calculator.costFor")
	defer span.End()
	span.SetAttributes(
		attribute.String("c3x.resource.kind", r.Ref.Kind),
		attribute.String("c3x.resource.name", r.Ref.Name),
	)
	def := e.registry.Get(r.Ref.Kind)
	if def == nil {
		e.logger.Debug("unknown resource kind; skipping costs",
			"kind", r.Ref.Kind, "name", r.Ref.Name)
		return domain.Cost{
			Resource: r.Ref,
			Currency: e.currency,
			Action:   r.Action,
		}, "unsupported kind (no catalog entry)", nil
	}

	lookup := e.priceLookupFor(ctx, r, def)

	var items []domain.LineItem
	subtotal := decimal.Zero

	for _, dim := range def.Dimensions {
		env := expr.EnvFor(r, lookup, dim.Constants)

		// `when` predicate (optional).
		if dim.When != "" {
			whenProg, err := e.programs.compile("when:"+def.Kind+"."+dim.ID, dim.When)
			if err != nil {
				return domain.Cost{}, "", err
			}
			ok, err := expr.RunBool(whenProg, env)
			if err != nil {
				return domain.Cost{}, "", fmt.Errorf("dimension %q: when: %w", dim.ID, err)
			}
			if !ok {
				continue
			}
		}

		quantityProg, err := e.programs.compile("qty:"+def.Kind+"."+dim.ID, dim.Quantity)
		if err != nil {
			return domain.Cost{}, "", err
		}
		qtyF, err := expr.RunNumber(quantityProg, env)
		if err != nil {
			return domain.Cost{}, "", fmt.Errorf("dimension %q: quantity: %w", dim.ID, err)
		}
		quantity := decimal.NewFromFloat(qtyF)

		rateProg, err := e.programs.compile("rate:"+def.Kind+"."+dim.ID, dim.Rate)
		if err != nil {
			return domain.Cost{}, "", err
		}
		rateF, err := expr.RunNumber(rateProg, env)
		if err != nil {
			return domain.Cost{}, "", fmt.Errorf("dimension %q: rate: %w", dim.ID, err)
		}
		rate := decimal.NewFromFloat(rateF)
		monthly := quantity.Mul(rate).Round(4)
		subtotal = subtotal.Add(monthly)

		items = append(items, domain.LineItem{
			Dimension:   dim.ID,
			Description: dim.Label,
			Unit:        dim.Unit,
			Quantity:    quantity,
			UnitRate:    rate,
			MonthlyCost: monthly,
			PriceSource: priceSourceFor(dim, lookup),
		})
	}

	// If we walked every dimension but emitted no priced line
	// items (and the kind isn't a known-free shell), report a
	// skip reason so --show-skipped can call this out.
	skipReason := ""
	if len(items) == 0 && !e.registry.IsFreeShell(def.Kind) && !catalog.IsLegitimatelyFree(def.Kind) {
		skipReason = "every dimension was guarded out (check `when` predicates and resource attributes)"
	}

	return domain.Cost{
		Resource:        r.Ref,
		LineItems:       items,
		MonthlySubtotal: subtotal.Round(2),
		Currency:        e.currency,
		Action:          r.Action,
	}, skipReason, nil
}

// priceLookupFor returns the closure injected into `price("...")`
// expression calls. The closure carries the per-resource context so the
// pricing.Query is constructed correctly.
func (e *Engine) priceLookupFor(
	ctx context.Context,
	r domain.Resource,
	def *catalog.Definition,
) expr.PriceLookup {
	region := r.ResolveRegion(e.defaultRegion)
	return func(mappingName string) (decimal.Decimal, string, error) {
		mapping, ok := def.Mappings[mappingName]
		if !ok {
			return decimal.Zero, domain.PriceSourceLive,
				fmt.Errorf("mapping %q not declared on %s", mappingName, def.Kind)
		}
		query, err := e.buildQuery(r, def, mapping, region)
		if err != nil {
			return decimal.Zero, domain.PriceSourceLive, err
		}
		rate, src, err := e.prices.Lookup(ctx, query)
		if err != nil {
			return decimal.Zero, domain.PriceSourceLive,
				fmt.Errorf("looking up %s: %w", mappingName, err)
		}
		return rate, mapPriceSource(src), nil
	}
}

// buildQuery materialises a pricing.Query from a catalog mapping by
// resolving any `expr` filter values against the resource's attributes.
func (e *Engine) buildQuery(
	r domain.Resource,
	def *catalog.Definition,
	m catalog.PriceMapping,
	region string,
) (pricing.Query, error) {
	if m.Region != "" {
		region = m.Region
	}
	filters := make([]pricing.KV, 0, len(m.AttributeFilters))
	for _, af := range m.AttributeFilters {
		value, err := resolveFilter(af, r, e.programs, def.Kind)
		if err != nil {
			return pricing.Query{}, err
		}
		filters = append(filters, pricing.KV{Key: af.Key, Value: value})
	}
	q := pricing.Query{
		Provider:         def.Provider,
		Service:          m.Service,
		ProductFamily:    m.ProductFamily,
		Region:           region,
		AttributeFilters: filters,
		PurchaseOption:   m.PurchaseOption,
		Unit:             m.Unit,
	}
	if q.PurchaseOption == "" {
		q.PurchaseOption = pricing.DefaultPurchaseOption(def.Provider)
	}
	return q, nil
}

// resolveFilter turns an AttributeFilter into a concrete value string.
// Literal short-circuits; Expr is compiled (cached) and run against
// the resource.
func resolveFilter(
	af catalog.AttributeFilter,
	r domain.Resource,
	cache *programCache,
	kind string,
) (string, error) {
	if af.Expr == "" {
		return af.Literal, nil
	}
	// Cache key includes the expression source so two mappings on the
	// same kind that filter on the same attribute key with different
	// expressions don't collide (e.g. ElastiCache Serverless's `data`
	// and `ecpu` mappings both filter `usagetype` but compute
	// different usagetype strings — without the source in the key,
	// the second compile() lookup returns the first mapping's
	// compiled program).
	prog, err := cache.compile("filter:"+kind+"."+af.Key+":"+af.Expr, af.Expr)
	if err != nil {
		return "", err
	}
	env := expr.EnvFor(r, nil, nil)
	return expr.RunString(prog, env)
}

// priceSourceFor decides what label to attach to a LineItem so the
// renderer/verifier can tell static-rate items apart. The lookup closure
// reports back via mapPriceSource; for dimensions whose Rate doesn't
// invoke price() at all (inline literals) we mark static.
func priceSourceFor(dim catalog.DimensionSpec, _ expr.PriceLookup) string {
	if !invokesPrice(dim.Rate) {
		return domain.PriceSourceStatic
	}
	return domain.PriceSourceLive
}

func invokesPrice(rate string) bool {
	for i := 0; i+6 <= len(rate); i++ {
		if rate[i:i+6] == "price(" {
			return true
		}
	}
	return false
}

// mapPriceSource normalises labels coming back from different Source
// implementations into the domain.PriceSource* constants.
func mapPriceSource(s string) string {
	switch s {
	case domain.PriceSourceLive, domain.PriceSourceStatic, domain.PriceSourceStub:
		return s
	default:
		return domain.PriceSourceLive
	}
}
