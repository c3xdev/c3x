package azure_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestLogAnalyticsWorkspaceGoldenFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tftest.GoldenFileResourceTestsWithOpts(t, "log_analytics_workspace_test", &tftest.GoldenFileOptions{
		CaptureLogs: true,
		IgnoreCLI:   true, // Azure no longer supports Standard/Premium SKUs
	})
}
