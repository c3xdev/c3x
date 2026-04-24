package aws_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestLambdaFunctionGoldenFile(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tftest.GoldenFileResourceTests(t, "lambda_function_test")
}

func TestLambdaFunctionChinaGoldenFile(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tftest.GoldenFileResourceTestsWithOpts(t, "lambda_function_china_test", &tftest.GoldenFileOptions{
		Currency: "CNY",
	})
}
