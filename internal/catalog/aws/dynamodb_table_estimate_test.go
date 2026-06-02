package aws_test

import (
	"testing"
	"time"

	resources "github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/stretchr/testify/assert"
)

func stubDynamoDBDescribeTable(stub *stubbedAWS) {
	stub.WhenBody(`{"TableName":""}`).Then(200, `{
    "Table": {
        "AttributeDefinitions": [],
        "TableName": "stubbed",
        "KeySchema": [],
        "TableStatus": "ACTIVE",
        "CreationDateTime": 0,
        "ProvisionedThroughput": {
            "NumberOfDecreasesToday": 0,
            "ReadCapacityUnits": 0,
            "WriteCapacityUnits": 0
        },
        "TableSizeBytes": 10737418240,
        "ItemCount": 1000,
        "TableArn": "arn:aws:dynamodb:us-east-2:012345678901:table/foo",
        "TableId": "00000000-0000-0000-0000-000000000000"
    }
	}`)
}

func TestDynamoDBStorage(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()
	stubDynamoDBDescribeTable(stub)

	args := resources.DynamoDBTable{}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)

	assert.Equal(t, int64(10), estimates.usage["storage_gb"])
}

func TestDynamoDBPayPerRequest(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()
	stubDynamoDBDescribeTable(stub)
	stub.WhenBody("ConsumedReadCapacityUnits", "Sum", "Count").
		OnPathPrefix(cloudWatchSmithyPathPrefix + "GetMetricStatistics").
		ThenCBOR(200, map[string]interface{}{
			"Label": "ConsumedReadCapacityUnits",
			"Datapoints": []interface{}{
				map[string]interface{}{
					"Unit":      "Count",
					"Sum":       122.6,
					"Timestamp": time.Unix(0, 0).UTC(),
				},
			},
		})
	stub.WhenBody("ConsumedWriteCapacityUnits", "Sum", "Count").
		OnPathPrefix(cloudWatchSmithyPathPrefix + "GetMetricStatistics").
		ThenCBOR(200, map[string]interface{}{
			"Label": "ConsumedWriteCapacityUnits",
			"Datapoints": []interface{}{
				map[string]interface{}{
					"Unit":      "Count",
					"Sum":       455.9,
					"Timestamp": time.Unix(0, 0).UTC(),
				},
			},
		})

	args := resources.DynamoDBTable{
		BillingMode: "PAY_PER_REQUEST",
	}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)

	assert.Equal(t, int64(123), estimates.usage["monthly_read_request_units"])
	assert.Equal(t, int64(456), estimates.usage["monthly_write_request_units"])
}

func TestDynamoDBProvisioned(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()
	stubDynamoDBDescribeTable(stub)

	args := resources.DynamoDBTable{
		BillingMode: "PROVISIONED",
	}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)

	assert.Equal(t, nil, estimates.usage["monthly_read_request_units"])
	assert.Equal(t, nil, estimates.usage["monthly_write_request_units"])
}
