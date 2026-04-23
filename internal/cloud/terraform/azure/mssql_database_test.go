package azure_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/settings"
	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestMSSQLDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	opts := tftest.DefaultGoldenFileOptions()
	opts.CaptureLogs = true
	opts.IgnoreCLI = true

	t.Run("base price", func(t *testing.T) {
		tftest.GoldenFileResourceTestsWithOpts(t, "mssql_database_test", opts)
	})

	t.Run("dev/test price", func(t *testing.T) {
		opts.GoldenFileSuffix = "dev_test_price"
		tftest.GoldenFileResourceTestsWithOpts(t, "mssql_database_test", opts, func(ctx *settings.Session) {
			ctx.Config.Projects[0].Metadata = map[string]string{
				"isProduction": "false",
			}
		})
	})
}

func TestMSSQLDatabaseWithBlankLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	opts := tftest.DefaultGoldenFileOptions()
	opts.CaptureLogs = true

	tftest.GoldenFileHCLResourceTestsWithOpts(t, "mssql_database_test_with_blank_location", opts)
}
