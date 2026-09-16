package pricing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// enumPageSize is the per-request product window. It matches the
// server's product cap so each page is one round-trip, and — crucially
// — every request stays bounded. The v1 failure mode (one ~8 GB
// streamed JSON that truncated into "unexpected end of JSON input")
// cannot recur here: a stalled or partial page fails that page only
// and retries cheaply.
const enumPageSize = 5000

// enumProduct is one product from the enumeration query — just enough
// to reconstruct the engine's query key (attributes) and resolve the
// rate locally (prices with their purchaseOption/unit), so warming the
// cache needs no second round-trip per product.
type enumProduct struct {
	ProductFamily string
	Attributes    map[string]string
	Prices        []enumPrice
}

type enumPrice struct {
	USD            string
	Unit           string
	PurchaseOption string
}

// productEnumerator pages through every product for a (provider,
// service, region) tuple via limit/offset pagination.
type productEnumerator struct {
	client   *http.Client
	endpoint string
	token    string
}

func newProductEnumerator(client *http.Client, endpoint, token string) *productEnumerator {
	if client == nil {
		client = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &productEnumerator{client: client, endpoint: endpoint, token: token}
}

// each invokes fn for every product in (provider, service, region),
// paging until a short page signals the end. Context cancellation
// stops the walk between pages.
func (e *productEnumerator) each(
	ctx context.Context,
	provider, service, region string,
	fn func(enumProduct) error,
) error {
	for offset := 0; ; offset += enumPageSize {
		if err := ctx.Err(); err != nil {
			return err
		}
		page, err := e.fetchPage(ctx, provider, service, region, offset)
		if err != nil {
			return err
		}
		for _, p := range page {
			if err := fn(p); err != nil {
				return err
			}
		}
		if len(page) < enumPageSize {
			return nil
		}
	}
}

func (e *productEnumerator) fetchPage(
	ctx context.Context,
	provider, service, region string,
	offset int,
) ([]enumProduct, error) {
	doc := buildEnumQuery(provider, service, region, enumPageSize, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint,
		bytes.NewReader([]byte(`{"query":`+jsonQuote(doc)+`}`)))
	if err != nil {
		return nil, fmt.Errorf("build enum request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "c3x/dev (+https://github.com/c3xdev/c3x)")
	if e.token != "" {
		req.Header.Set("Authorization", "Bearer "+e.token)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", e.endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		preview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("upstream %s returned %d: %s",
			e.endpoint, resp.StatusCode, bytes.TrimSpace(preview))
	}

	var decoded enumResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode enum response: %w", err)
	}
	if len(decoded.Errors) > 0 {
		msgs := make([]string, len(decoded.Errors))
		for i, m := range decoded.Errors {
			msgs[i] = m.Message
		}
		return nil, fmt.Errorf("upstream errors: %s", strings.Join(msgs, "; "))
	}

	out := make([]enumProduct, 0, len(decoded.Data.Products))
	for _, p := range decoded.Data.Products {
		attrs := make(map[string]string, len(p.Attributes))
		for _, a := range p.Attributes {
			attrs[a.Key] = a.Value
		}
		prices := make([]enumPrice, 0, len(p.Prices))
		for _, pr := range p.Prices {
			prices = append(prices, enumPrice(pr))
		}
		out = append(out, enumProduct{ProductFamily: p.ProductFamily, Attributes: attrs, Prices: prices})
	}
	return out, nil
}

// buildEnumQuery renders the enumeration GraphQL document. Region "" or
// "global" omits the region filter (matching how the live path treats
// non-regional services).
func buildEnumQuery(provider, service, region string, limit, offset int) string {
	var b strings.Builder
	b.WriteString(`{products(filter:{vendorName:"`)
	b.WriteString(escape(provider))
	b.WriteString(`",service:"`)
	b.WriteString(escape(service))
	b.WriteString(`"`)
	if region != "" && region != "global" {
		b.WriteString(`,region:"`)
		b.WriteString(escape(region))
		b.WriteString(`"`)
	}
	fmt.Fprintf(&b, `},limit:%d,offset:%d)`, limit, offset)
	b.WriteString(`{productFamily attributes{key value} prices{USD unit purchaseOption}}}`)
	return b.String()
}

type enumResponse struct {
	Data   enumData        `json:"data"`
	Errors []graphqlErrMsg `json:"errors,omitempty"`
}

type enumData struct {
	Products []enumProductRaw `json:"products"`
}

type enumProductRaw struct {
	ProductFamily string         `json:"productFamily"`
	Attributes    []enumAttrRaw  `json:"attributes"`
	Prices        []enumPriceRaw `json:"prices"`
}

type enumAttrRaw struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type enumPriceRaw struct {
	USD            string `json:"USD"`
	Unit           string `json:"unit"`
	PurchaseOption string `json:"purchaseOption"`
}
