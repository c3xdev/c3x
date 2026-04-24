package main_test

import (
	"github.com/c3xdev/c3x/internal/testutil"
	"testing"
)

func TestAuthLoginHelpFlag(t *testing.T) {
	GoldenFileCommandTest(t, testutil.CalcGoldenFileTestdataDirName(), []string{"auth", "login", "--help"}, nil)
}
