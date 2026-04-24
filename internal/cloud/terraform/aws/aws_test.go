package aws_test

import (
	"os"
	"testing"
	"time"

	aws "github.com/c3xdev/c3x/internal/catalog/aws"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestMain(m *testing.M) {
	// Pin the date used for extended support pricing so tests are deterministic.
	aws.Today = time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC)

	tftest.EnsurePluginsInstalled()
	code := m.Run()
	os.Exit(code)
}
