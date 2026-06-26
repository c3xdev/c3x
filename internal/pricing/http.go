package pricing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/c3xdev/c3x/internal/domain"
	"github.com/c3xdev/c3x/internal/observability"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
)

// DefaultEndpoint is the canonical c3x pricing API. The HTTP client
// defaults to it; users can override via [config.Resolved.PricingEndpoint]
// or `--pricing-endpoint`.
const DefaultEndpoint = "https://pricing.c3x.dev/graphql"

// DefaultHTTPTimeout caps a single GraphQL request. Lookups that take
// longer almost always indicate an upstream problem; failing fast and
// surfacing the warning beats a hung c3x run.
const DefaultHTTPTimeout = 30 * time.Second

// HTTPSource talks GraphQL to pricing.c3x.dev (or any compatible
// endpoint exposing the same schema). Construct via [NewHTTPSource];
// the zero value is not usable.
//
// Concurrency: one HTTPSource may be shared across goroutines. The
// underlying [http.Client] reuses connections via its Transport.
type HTTPSource struct {
	endpoint string
	client   *http.Client
}

// HTTPOption configures an [HTTPSource].
type HTTPOption func(*HTTPSource)

// WithEndpoint overrides the GraphQL endpoint URL.
func WithEndpoint(url string) HTTPOption { return func(s *HTTPSource) { s.endpoint = url } }

// WithHTTPClient lets tests inject a custom *http.Client (typically
// pointing at an httptest.Server). Production callers should pass nil
// and let the default client handle pooling.
func WithHTTPClient(c *http.Client) HTTPOption {
	return func(s *HTTPSource) {
		if c != nil {
			s.client = c
		}
	}
}

// NewHTTPSource returns a configured HTTPSource. Endpoint defaults to
// [DefaultEndpoint]; HTTP client uses a 30s timeout and the stdlib
// pooling Transport.
func NewHTTPSource(opts ...HTTPOption) *HTTPSource {
	s := &HTTPSource{
		endpoint: DefaultEndpoint,
		client:   &http.Client{Timeout: DefaultHTTPTimeout},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Endpoint returns the configured URL — exposed so the CLI can echo it
// during `--verbose` runs.
func (s *HTTPSource) Endpoint() string { return s.endpoint }

// Lookup implements [Source]. It builds a GraphQL document from the
// Query, POSTs it to the endpoint, and returns the first non-zero
// USD price from the first matching product. A zero return with nil
// error means the query matched no priced products — legitimate for
// always-free resources (ACM public certificates, ALB target groups).
//
// Tiered prices: the upstream may return several prices for one
// product (e.g. SNS HTTP deliveries quote $0 free-tier first, then the
// paid rate). We skip zeros and prefer the first non-zero entry,
// falling back to zero only if every price is zero.
func (s *HTTPSource) Lookup(ctx context.Context, q Query) (decimal.Decimal, string, error) {
	rate, src, err := s.lookupOnce(ctx, q)
	if err != nil || !rate.IsZero() {
		return rate, src, err
	}
	// Reference-region fallback. Many catalog filters pin SKU strings
	// that only exist under the provider's reference region (AWS
	// usagetypes carry a "USE1-" prefix, GCP descriptions embed
	// "Iowa", Azure meters are densest under eastus). A regional miss
	// would otherwise silently produce $0 — the worst failure mode a
	// cost tool can have. Quoting the reference region's rate is the
	// honest degradation: regional deltas are single-digit percent
	// for most services, while $0 is infinitely wrong.
	if ref := referenceRegion(q.Provider); ref != "" && q.Region != ref &&
		q.Region != "" && q.Region != "global" {
		fq := q
		fq.Region = ref
		frate, fsrc, ferr := s.lookupOnce(ctx, fq)
		if ferr == nil && !frate.IsZero() {
			return frate, fsrc, nil
		}
	}
	return rate, src, err
}

// referenceRegion is the region whose SKUs the catalog's attribute
// filters are written against — the fallback when a regional lookup
// matches nothing.
func referenceRegion(provider string) string {
	switch provider {
	case "aws":
		return "us-east-1"
	case "azure":
		return "eastus"
	case "gcp":
		return "us-central1"
	default:
		return ""
	}
}

func (s *HTTPSource) lookupOnce(ctx context.Context, q Query) (decimal.Decimal, string, error) {
	ctx, span := observability.Tracer().Start(ctx, "pricing.HTTPSource.Lookup")
	defer span.End()
	span.SetAttributes(
		attribute.String("c3x.provider", q.Provider),
		attribute.String("c3x.service", q.Service),
		attribute.String("c3x.region", q.Region),
	)
	body := buildQuery(q)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint,
		bytes.NewReader([]byte(`{"query":`+jsonQuote(body)+`}`)))
	if err != nil {
		return decimal.Zero, domain.PriceSourceLive, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "c3x/dev (+https://github.com/c3xdev/c3x)")

	resp, err := s.client.Do(req)
	if err != nil {
		return decimal.Zero, domain.PriceSourceLive, fmt.Errorf("POST %s: %w", s.endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// Read a small slice of the body for diagnostics, then bail.
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return decimal.Zero, domain.PriceSourceLive,
			fmt.Errorf("upstream %s returned %d: %s",
				s.endpoint, resp.StatusCode, bytes.TrimSpace(preview))
	}

	var decoded graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return decimal.Zero, domain.PriceSourceLive, fmt.Errorf("decode response: %w", err)
	}
	if len(decoded.Errors) > 0 {
		msgs := make([]string, len(decoded.Errors))
		for i, e := range decoded.Errors {
			msgs[i] = e.Message
		}
		return decimal.Zero, domain.PriceSourceLive,
			fmt.Errorf("upstream errors: %s", strings.Join(msgs, "; "))
	}

	rate, err := pickNonZeroPrice(decoded.Data)
	if err != nil {
		return decimal.Zero, domain.PriceSourceLive, err
	}
	return rate, domain.PriceSourceLive, nil
}

// buildQuery renders the GraphQL document for a Query.
//
// Special cases:
//   - Region "" or "global" → omits the region filter (matches
//     services like CloudFront that aren't per-region).
//   - Empty ProductFamily → omits the productFamily filter so we don't
//     accidentally constrain to the empty-string family.
//   - PurchaseOption / Unit → applied as nested filters on the prices
//     sub-field, so unrelated price-array entries don't pollute the
//     non-zero pick.
func buildQuery(q Query) string {
	var b strings.Builder
	b.WriteString(`{products(filter:{vendorName:"`)
	b.WriteString(escape(q.Provider))
	b.WriteString(`",service:"`)
	b.WriteString(escape(q.Service))
	b.WriteString(`"`)
	if q.ProductFamily != "" {
		b.WriteString(`,productFamily:"`)
		b.WriteString(escape(q.ProductFamily))
		b.WriteString(`"`)
	}
	if q.Region != "" && q.Region != "global" {
		b.WriteString(`,region:"`)
		b.WriteString(escape(q.Region))
		b.WriteString(`"`)
	}
	if len(q.AttributeFilters) > 0 {
		b.WriteString(`,attributeFilters:[`)
		for i, f := range q.AttributeFilters {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(`{key:"`)
			b.WriteString(escape(f.Key))
			b.WriteString(`",value:"`)
			b.WriteString(escape(f.Value))
			b.WriteString(`"}`)
		}
		b.WriteString(`]`)
	}
	// limit:50 (not 1): some product families return multiple
	// variants where only one has the desired purchaseOption (e.g.
	// EC2 Dedicated Host has separate products for No-Upfront-RI and
	// On-Demand; limit:1 may land on the variant with empty
	// on_demand prices). The max-non-zero picker then chooses across
	// the returned products. 50 stays well under the upstream's
	// 5000-product cap and keeps the response small for the common
	// case where filters already narrow to 1.
	b.WriteString(`},limit:50){prices`)
	// PurchaseOption "any" is a catalog-side sentinel meaning the
	// upstream's prices for this product carry no purchaseOption
	// label, so we omit that filter clause entirely. Without it the
	// subfield filter would match zero rows and the rate would be 0.
	includePO := q.PurchaseOption != "" && q.PurchaseOption != "any"
	if includePO || q.Unit != "" {
		b.WriteString(`(filter:{`)
		first := true
		if includePO {
			b.WriteString(`purchaseOption:"`)
			b.WriteString(escape(q.PurchaseOption))
			b.WriteString(`"`)
			first = false
		}
		if q.Unit != "" {
			if !first {
				b.WriteString(",")
			}
			b.WriteString(`unit:"`)
			b.WriteString(escape(q.Unit))
			b.WriteString(`"`)
		}
		b.WriteString(`})`)
	}
	b.WriteString(`{USD unit}}}`)
	return b.String()
}

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// jsonQuote wraps a string for a JSON body. We render the GraphQL
// document as a string field of `{"query": "..."}` without using
// encoding/json on the outer envelope because the query text is
// trusted (built by us) and avoiding json.Marshal on the hot path
// keeps the per-lookup CPU profile flat.
func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// pickNonZeroPrice walks the (products, prices) tree and returns the
// **largest** non-zero USD value, or zero if every price was zero.
//
// Why max and not first: tiered services (CloudFront data transfer,
// S3 requests) return several prices per product — typically a mix
// of committed-tier rates (cheaper) and on-demand tier rates (more
// expensive). Picking "first non-zero" landed on whichever the
// upstream put first, which historically meant the cheapest
// committed rate — silently understating cost by 3–4×. Picking the
// maximum non-zero is the conservative choice: it surfaces the
// first-tier on-demand rate a brand-new workload actually pays.
//
// Single-rate products (most resources) are unaffected: max of one
// value is that value. Free-tier patterns like SNS ($0 free, then
// $0.0000006 paid) still work: the zero is skipped and the paid rate
// wins.
func pickNonZeroPrice(data *graphqlData) (decimal.Decimal, error) {
	if data == nil {
		return decimal.Zero, nil
	}
	maxNonZero := decimal.Zero
	haveNonZero := false
	for _, p := range data.Products {
		for _, price := range p.Prices {
			if price.USD == "" {
				continue
			}
			d, err := decimal.NewFromString(price.USD)
			if err != nil {
				return decimal.Zero, fmt.Errorf("invalid price %q: %w", price.USD, err)
			}
			if d.IsZero() {
				continue
			}
			if !haveNonZero || d.GreaterThan(maxNonZero) {
				maxNonZero = d
				haveNonZero = true
			}
		}
	}
	return maxNonZero, nil
}

// PriceSpread describes the distribution of non-zero USD prices a query
// matched. A wide spread (Max ≫ MinNonZero) on a Count>1 query is the
// signature of an under-specified mapping: the max-non-zero picker is
// likely selecting a premium SKU (provisioned, Multi-AZ, IO-optimized)
// rather than the intended one. Used by the `verify_catalog --audit`
// sweep, not the estimate hot path.
type PriceSpread struct {
	MinNonZero decimal.Decimal
	Max        decimal.Decimal
	Count      int // distinct non-zero prices matched
}

// Spread runs the same GraphQL query as [HTTPSource.Lookup] but reports
// the full distribution of matching non-zero prices instead of just the
// max. It exists for catalog auditing.
func (s *HTTPSource) Spread(ctx context.Context, q Query) (PriceSpread, error) {
	body := buildQuery(q)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint,
		bytes.NewReader([]byte(`{"query":`+jsonQuote(body)+`}`)))
	if err != nil {
		return PriceSpread{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "c3x/dev (+https://github.com/c3xdev/c3x)")
	resp, err := s.client.Do(req)
	if err != nil {
		return PriceSpread{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return PriceSpread{}, fmt.Errorf("upstream %d", resp.StatusCode)
	}
	var decoded graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return PriceSpread{}, err
	}
	var sp PriceSpread
	seen := map[string]bool{}
	if decoded.Data != nil {
		for _, p := range decoded.Data.Products {
			for _, price := range p.Prices {
				if price.USD == "" || seen[price.USD] {
					continue
				}
				d, derr := decimal.NewFromString(price.USD)
				if derr != nil || d.IsZero() {
					continue
				}
				seen[price.USD] = true
				if sp.Count == 0 || d.LessThan(sp.MinNonZero) {
					sp.MinNonZero = d
				}
				if d.GreaterThan(sp.Max) {
					sp.Max = d
				}
				sp.Count++
			}
		}
	}
	return sp, nil
}

// GraphQL response shape. We only declare the fields we read so the
// JSON decoder is forgiving about additions on the server side.
type graphqlResponse struct {
	Data   *graphqlData    `json:"data,omitempty"`
	Errors []graphqlErrMsg `json:"errors,omitempty"`
}

type graphqlData struct {
	Products []graphqlProduct `json:"products"`
}

type graphqlProduct struct {
	Prices []graphqlPrice `json:"prices"`
}

type graphqlPrice struct {
	USD  string `json:"USD"`
	Unit string `json:"unit"`
}

type graphqlErrMsg struct {
	Message string `json:"message"`
}
