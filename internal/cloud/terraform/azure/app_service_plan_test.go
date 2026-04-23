package azure_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/settings"
	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestAzureRMAppServicePlan(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	t.Run("base price", func(t *testing.T) {
		tftest.GoldenFileResourceTestsWithOpts(t, "app_service_plan_test", &tftest.GoldenFileOptions{
			IgnoreCLI: true,
		})
	})

	t.Run("dev/test price", func(t *testing.T) {
		tftest.GoldenFileResourceTestsWithOpts(t, "app_service_plan_test", &tftest.GoldenFileOptions{
			GoldenFileSuffix: "dev_test_price",
			IgnoreCLI:        true,
		}, func(ctx *settings.Session) {
			ctx.Config.Projects[0].Metadata = map[string]string{
				"isProduction": "false",
			}
		})
	})
}
