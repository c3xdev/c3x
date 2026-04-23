package google_test

import (
	"os"
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestMain(m *testing.M) {
	tftest.EnsurePluginsInstalled()
	code := m.Run()
	os.Exit(code)
}
