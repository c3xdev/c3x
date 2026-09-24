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

// regionalServer has prices only for us-east-1, like a catalog filter that
// pins a reference-region SKU.
func regionalServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(raw), "us-east-1") {
			_, _ = w.Write([]byte(`{"data":{"products":[{"prices":[{"USD":"0.0225","unit":"Hrs"}]}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"products":[]}}`))
	}))
}

func TestHTTPSourceMarksReferenceRegionFallback(t *testing.T) {
	t.Parallel()
	srv := regionalServer(t)
	t.Cleanup(srv.Close)
	src := pricing.NewHTTPSource(pricing.WithEndpoint(srv.URL), pricing.WithHTTPClient(srv.Client()))
	rate, s, err := src.Lookup(context.Background(), pricing.Query{Provider: "aws", Service: "AWSELB", Region: "sa-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	if rate.String() != "0.0225" {
		t.Errorf("rate = %s, want the reference-region rate", rate)
	}
	if ref, ok := pricing.SourceFlag(s, pricing.FlagFallback); !ok || ref != "us-east-1" {
		t.Errorf("source %q must carry fallback=us-east-1", s)
	}
}

func TestHTTPSourceDirectRegionalHitHasNoMarker(t *testing.T) {
	t.Parallel()
	srv := regionalServer(t)
	t.Cleanup(srv.Close)
	src := pricing.NewHTTPSource(pricing.WithEndpoint(srv.URL), pricing.WithHTTPClient(srv.Client()))
	_, s, err := src.Lookup(context.Background(), pricing.Query{Provider: "aws", Service: "AWSELB", Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	if s != "live" {
		t.Errorf("source = %q, want plain live", s)
	}
}

func TestHTTPSourceMarksNoMatch(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"products":[]}}`))
	}))
	t.Cleanup(srv.Close)
	src := pricing.NewHTTPSource(pricing.WithEndpoint(srv.URL), pricing.WithHTTPClient(srv.Client()))
	_, s, err := src.Lookup(context.Background(), pricing.Query{Provider: "aws", Service: "AmazonEC2", Region: "us-east-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pricing.SourceFlag(s, pricing.FlagNoMatch); !ok {
		t.Errorf("source %q must carry nomatch", s)
	}
}
