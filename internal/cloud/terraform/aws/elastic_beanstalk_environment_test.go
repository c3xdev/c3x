package aws_test

import (
	"testing"

	"github.com/c3xdev/c3x/internal/cloud/terraform/tftest"
)

func TestElasticBeanstalkEnvironmentGoldenFile(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	opts := tftest.DefaultGoldenFileOptions()
	opts.CaptureLogs = true
	tftest.GoldenFileResourceTestsWithOpts(t, "elastic_beanstalk_environment_test", opts)

}
