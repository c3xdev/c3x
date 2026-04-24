package aws_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestCloudHSMv2HSM(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tftest.GoldenFileResourceTests(t, "cloudhsm_v2_hsm_test")
}
