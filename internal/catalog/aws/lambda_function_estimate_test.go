package aws_test

import (
	"testing"
	"time"

	resources "github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/stretchr/testify/assert"
)

func TestLambda(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()

	stub.WhenBody("Invocations", "Sum").
		OnPathPrefix(cloudWatchSmithyPathPrefix+"GetMetricStatistics").
		ThenCBOR(200, map[string]interface{}{
			"Label": "Invocations",
			"Datapoints": []interface{}{
				map[string]interface{}{
					"Unit":      "Count",
					"Sum":       1234.0,
					"Timestamp": time.Unix(0, 0).UTC(),
				},
			},
		})
	stub.WhenBody("Duration", "Average").
		OnPathPrefix(cloudWatchSmithyPathPrefix+"GetMetricStatistics").
		ThenCBOR(200, map[string]interface{}{
			"Label": "Duration",
			"Datapoints": []interface{}{
				map[string]interface{}{
					"Average":   5678.9,
					"Unit":      "Milliseconds",
					"Timestamp": time.Unix(0, 0).UTC(),
				},
			},
		})

	args := &resources.LambdaFunction{}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)
	assert.Equal(t, int64(1234), estimates.usage["monthly_requests"])
	assert.Equal(t, int64(5679), estimates.usage["request_duration_ms"])
}
