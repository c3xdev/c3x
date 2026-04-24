package azure_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestAzureRMCDNEndpoint(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tftest.GoldenFileResourceTestsWithOpts(t, "cdn_endpoint_test", &tftest.GoldenFileOptions{
		IgnoreCLI: true, // the creation of new CDN resources is no longer permitted
	})
}
