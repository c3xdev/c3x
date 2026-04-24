package main_test

import (
	"github.com/c3xdev/c3x/internal/testutil"
	"testing"
)

func TestNoArgs(t *testing.T) {
	GoldenFileCommandTest(t, testutil.CalcGoldenFileTestdataDirName(), []string{}, nil)
}

func TestHelpFlag(t *testing.T) {
	GoldenFileCommandTest(t, testutil.CalcGoldenFileTestdataDirName(), []string{"--help"}, nil)
}

func TestHelp(t *testing.T) {
	GoldenFileCommandTest(t, testutil.CalcGoldenFileTestdataDirName(), []string{"help"}, nil)
}

func TestHelpHelp(t *testing.T) {
	GoldenFileCommandTest(t, testutil.CalcGoldenFileTestdataDirName(), []string{"help", "--help"}, nil)
}
