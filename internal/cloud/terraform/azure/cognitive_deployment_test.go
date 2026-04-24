package azure_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestCognitiveDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	opts := tftest.DefaultGoldenFileOptions()
	opts.CaptureLogs = true
	opts.IgnoreCLI = true
	tftest.GoldenFileResourceTestsWithOpts(t, "cognitive_deployment_test", opts)
}
