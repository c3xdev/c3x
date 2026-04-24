package google_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestBigqueryTable(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tftest.GoldenFileResourceTests(t, "bigquery_table_test")
}
