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

// fixedServer prices exactly the (region, usagetype) pairs it is given.
func fixedServer(t *testing.T, prices map[[2]string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body := string(raw)
		for k, price := range prices {
			// The usagetype must match exactly, not as a substring of a
			// prefixed one.
			if strings.Contains(body, `\"`+k[0]+`\"`) && strings.Contains(body, `value:\"`+k[1]+`\"`) {
				_, _ = io.WriteString(w, `{"data":{"products":[{"prices":[{"USD":"`+price+`","unit":"Hrs"}]}]}}`)
				return
			}
		}
		_, _ = io.WriteString(w, `{"data":{"products":[]}}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func lookup(t *testing.T, srv *httptest.Server, region, usagetype string) (string, string) {
	t.Helper()
	src := pricing.NewHTTPSource(pricing.WithEndpoint(srv.URL), pricing.WithHTTPClient(srv.Client()))
	rate, s, err := src.Lookup(context.Background(), pricing.Query{
		Provider: "aws", Service: "AmazonRDS", Region: region,
		AttributeFilters: []pricing.KV{{Key: "usagetype", Value: usagetype}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return rate.String(), s
}

// us-east-1 writes some usagetypes with no prefix at all; other regions
// prefix them.
func TestUnprefixedUsagetypeIsLocalized(t *testing.T) {
	t.Parallel()
	srv := fixedServer(t, map[[2]string]string{
		{"us-east-1", "Aurora:ServerlessV2Usage"}:    "0.12",
		{"eu-west-1", "EU-Aurora:ServerlessV2Usage"}: "0.14",
	})
	rate, s := lookup(t, srv, "eu-west-1", "Aurora:ServerlessV2Usage")
	if rate != "0.14" {
		t.Errorf("rate = %s, want eu-west-1's 0.14", rate)
	}
	if _, ok := pricing.SourceFlag(s, pricing.FlagFallback); ok {
		t.Errorf("source %q: a regional hit must not be marked as a fallback", s)
	}
}

// A service whose usagetypes are unprefixed in every region still matches
// its own region: the rewrite misses and the original query is retried.
func TestUnprefixedEverywhereStillMatchesItsRegion(t *testing.T) {
	t.Parallel()
	srv := fixedServer(t, map[[2]string]string{
		{"us-east-1", "Requests-Tier1"}: "0.005",
		{"eu-west-1", "Requests-Tier1"}: "0.0054",
	})
	if rate, _ := lookup(t, srv, "eu-west-1", "Requests-Tier1"); rate != "0.0054" {
		t.Errorf("rate = %s, want eu-west-1's own 0.0054", rate)
	}
}

// An already-regional usagetype is left alone.
func TestPrefixedUsagetypeIsNotRewritten(t *testing.T) {
	t.Parallel()
	srv := fixedServer(t, map[[2]string]string{
		{"eu-west-1", "EU-DataTransfer-Out-Bytes"}: "0.09",
	})
	if rate, _ := lookup(t, srv, "eu-west-1", "EU-DataTransfer-Out-Bytes"); rate != "0.09" {
		t.Errorf("rate = %s, want 0.09", rate)
	}
}
