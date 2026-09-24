package pricing_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/pricing"
)

// usagetypeServer prices EKS the way the real data does: "EU-" in
// eu-west-1, "USE1-" in us-east-1, and nothing for any other pairing.
// euPriced=false models a region the data doesn't cover.
func usagetypeServer(t *testing.T, euPriced bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		price := ""
		switch {
		case strings.Contains(body, "us-east-1") && strings.Contains(body, "USE1-AmazonEKS-Hours"):
			price = "0.10"
		case euPriced && strings.Contains(body, "eu-west-1") && strings.Contains(body, "EU-AmazonEKS-Hours"):
			price = "0.11"
		}
		if price == "" {
			_, _ = io.WriteString(w, `{"data":{"products":[]}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"products":[{"prices":[{"USD":"`+price+`","unit":"Hrs"}]}]}}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func eksQuery(region string) pricing.Query {
	return pricing.Query{
		Provider: "aws", Service: "AmazonEKS", Region: region,
		AttributeFilters: []pricing.KV{{Key: "usagetype", Value: "USE1-AmazonEKS-Hours:perCluster"}},
	}
}

func TestUsagetypeIsLocalizedToTheRegion(t *testing.T) {
	t.Parallel()
	srv := usagetypeServer(t, true)
	src := pricing.NewHTTPSource(pricing.WithEndpoint(srv.URL), pricing.WithHTTPClient(srv.Client()))
	rate, s, err := src.Lookup(context.Background(), eksQuery("eu-west-1"))
	if err != nil {
		t.Fatal(err)
	}
	if rate.String() != "0.11" {
		t.Errorf("rate = %s, want eu-west-1's own 0.11 (was the us-east-1 fallback)", rate)
	}
	if _, ok := pricing.SourceFlag(s, pricing.FlagFallback); ok {
		t.Errorf("source %q: a regional hit must not be marked as a fallback", s)
	}
}

func TestLocalizedMissStillFallsBackWithOriginalFilters(t *testing.T) {
	t.Parallel()
	srv := usagetypeServer(t, false)
	src := pricing.NewHTTPSource(pricing.WithEndpoint(srv.URL), pricing.WithHTTPClient(srv.Client()))
	rate, s, err := src.Lookup(context.Background(), eksQuery("eu-west-1"))
	if err != nil {
		t.Fatal(err)
	}
	// The fallback must query us-east-1 with USE1-, not with the EU-
	// rewrite, or it would match nothing and quote $0.
	if rate.String() != "0.1" {
		t.Errorf("rate = %s, want the us-east-1 fallback 0.10", rate)
	}
	if ref, ok := pricing.SourceFlag(s, pricing.FlagFallback); !ok || ref != "us-east-1" {
		t.Errorf("source %q must carry fallback=us-east-1", s)
	}
}

func TestUsagetypeUntouchedInUSEast1(t *testing.T) {
	t.Parallel()
	srv := usagetypeServer(t, true)
	src := pricing.NewHTTPSource(pricing.WithEndpoint(srv.URL), pricing.WithHTTPClient(srv.Client()))
	rate, _, err := src.Lookup(context.Background(), eksQuery("us-east-1"))
	if err != nil || rate.String() != "0.1" {
		t.Errorf("us-east-1: rate = %s, err = %v", rate, err)
	}
}
