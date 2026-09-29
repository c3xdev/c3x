package pricing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// A server that caps each page below enumPageSize, as pricing.c3x.dev
// does with MAX_PRODUCTS_PER_REQUEST, must still be enumerated in full.
// Before the fix the enumerator treated the first short page as the last
// one, so `c3x pricing sync` kept only the first 1,000 products of each
// service and region.
func TestEnumeratorPagesThroughAServerCap(t *testing.T) {
	t.Parallel()
	const total, capPerPage = 7, 3
	offsetRe := regexp.MustCompile(`offset:(\d+)`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		m := offsetRe.FindStringSubmatch(string(raw))
		offset, _ := strconv.Atoi(m[1])
		var items []string
		for i := offset; i < total && i < offset+capPerPage; i++ {
			items = append(items, fmt.Sprintf(
				`{"productFamily":"Compute Instance","attributes":[{"key":"instanceType","value":"t%d"}],"prices":[{"USD":"0.1","unit":"Hrs","purchaseOption":"on_demand"}]}`, i))
		}
		_, _ = io.WriteString(w, `{"data":{"products":[`+strings.Join(items, ",")+`]}}`)
	}))
	t.Cleanup(srv.Close)

	enum := newProductEnumerator(srv.Client(), srv.URL, "")
	seen := map[string]bool{}
	err := enum.each(context.Background(), "aws", "AmazonEC2", "us-east-1", func(p enumProduct) error {
		seen[p.Attributes["instanceType"]] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != total {
		t.Fatalf("enumerated %d distinct products, want all %d: %v", len(seen), total, seen)
	}
}
