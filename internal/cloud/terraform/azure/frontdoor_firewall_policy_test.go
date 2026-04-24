package azure_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestFrontdoorFirewallPolicyGoldenFile(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tftest.GoldenFileResourceTestsWithOpts(t, "frontdoor_firewall_policy_test", &tftest.GoldenFileOptions{
		IgnoreCLI: true, // the creation of new Frontdoor resources is no longer permitted
	})
}
