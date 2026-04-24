package azure_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestSQLElasticPool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	opts := tftest.DefaultGoldenFileOptions()
	opts.CaptureLogs = true
	// ignore CLI - this has been removed from the latest provider
	opts.IgnoreCLI = true
	tftest.GoldenFileResourceTestsWithOpts(t, "sql_elasticpool_test", opts)
}
