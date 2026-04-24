package main_test

import (
	"github.com/c3xdev/c3x/internal/testutil"
	"testing"
)

// We don't have a great way to test this, so just test the help.

func TestRegisterHelpFlag(t *testing.T) {
	GoldenFileCommandTest(t, testutil.CalcGoldenFileTestdataDirName(), []string{"register", "--help"}, nil)
}
