package pricing_test

// Auth-header tests for the pricing HTTP client. A self-hosted
// c3x-pricing-api with API_KEY set accepts `Authorization: Bearer
// <key>`; the public endpoint needs no auth, so the header must be
// absent when no token is configured.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/c3xdev/c3x/internal/pricing"
)

const okPriceResponse = `{"data":{"products":[{"prices":[{"USD":"0.10","unit":"Hrs"}]}]}}`

func TestHTTPSourceSendsBearerTokenWhenSet(t *testing.T) {
	t.Parallel()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okPriceResponse))
	}))
	t.Cleanup(srv.Close)

	src := pricing.NewHTTPSource(
		pricing.WithEndpoint(srv.URL),
		pricing.WithToken("s3cr3t"),
		pricing.WithHTTPClient(srv.Client()),
	)
	if _, _, err := src.Lookup(context.Background(), pricing.Query{
		Provider: "aws", Service: "AmazonEC2", Region: "us-east-1",
	}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer s3cr3t" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer s3cr3t")
	}
}

func TestHTTPSourceOmitsAuthWhenNoToken(t *testing.T) {
	t.Parallel()
	var hadAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadAuth = r.Header["Authorization"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(okPriceResponse))
	}))
	t.Cleanup(srv.Close)

	src := pricing.NewHTTPSource(
		pricing.WithEndpoint(srv.URL),
		pricing.WithHTTPClient(srv.Client()),
	)
	if _, _, err := src.Lookup(context.Background(), pricing.Query{
		Provider: "aws", Service: "AmazonEC2", Region: "us-east-1",
	}); err != nil {
		t.Fatal(err)
	}
	if hadAuth {
		t.Error("no token configured, but an Authorization header was sent")
	}
}
