package azure_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestAzureRMCosmosDBCassandraTableGoldenFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	opts := tftest.DefaultGoldenFileOptions()
	opts.CaptureLogs = true
	tftest.GoldenFileResourceTestsWithOpts(t, "cosmosdb_cassandra_table_test", opts)
}
