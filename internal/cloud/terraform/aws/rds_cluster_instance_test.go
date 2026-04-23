package aws_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestRDSClusterInstanceGoldenFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tftest.GoldenFileResourceTestsWithOpts(t, "rds_cluster_instance_test", &tftest.GoldenFileOptions{
		CaptureLogs: true,
	})
}
