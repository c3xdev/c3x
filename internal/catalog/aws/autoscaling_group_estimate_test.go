package aws_test

import (
	"testing"
	"time"

	resources "github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/stretchr/testify/assert"
)

// stubCloudWatchASGQuery stubs a CloudWatch GetMetricStatistics call for the
// GroupTotalInstances metric on the named AutoScalingGroup. CloudWatch uses
// Smithy RPCv2 CBOR since service/cloudwatch v1.52+, so we match on the
// Smithy operation path and an ASG name fragment present in the CBOR body
// (text strings appear literally in CBOR-encoded bytes).
func stubCloudWatchASGQuery(stub *stubbedAWS, name string, value float64) {
	datapoints := []interface{}{}
	if value > 0 {
		datapoints = append(datapoints, map[string]interface{}{
			"Average":   value,
			"Unit":      "None",
			"Timestamp": time.Unix(0, 0).UTC(),
		})
	}
	stub.WhenBody(name, "AutoScalingGroupName", "GroupTotalInstances").
		OnPathPrefix(cloudWatchSmithyPathPrefix+"GetMetricStatistics").
		ThenCBOR(200, map[string]interface{}{
			"Datapoints": datapoints,
			"Label":      "GroupTotalInstances",
		})
}

func stubEC2DescribeAutoscalingGroups(stub *stubbedAWS, name string, count int64) {
	var instanceMembers string
	var groupMember string

	// shoddy stub: woefully incomplete compared to real response
	if count > 0 {
		for i := int64(0); i < count; i++ {
			instanceMembers += `
					<member></member>`
		}

		groupMember = `
			<member>
				<Instances>
				` + instanceMembers +
			`</Instances>
			</member>`
	}

	stub.WhenBody("Action=DescribeAutoScalingGroups&AutoScalingGroupNames.member.1="+name).Then(200, `
		<DescribeAutoScalingGroupsResponse xmlns="http://autoscaling.amazonaws.com/doc/2011-01-01/">
	  <DescribeAutoScalingGroupsResult>
	    <AutoScalingGroups>
		 	`+groupMember+`
	    </AutoScalingGroups>
	  </DescribeAutoScalingGroupsResult>
		<ResponseMetadata>
			<RequestId>00000000-0000-0000-0000-000000000000</RequestId>
		</ResponseMetadata>
	</DescribeAutoScalingGroupsResponse>
	`)
}

// Tests LaunchConfiguration as a side effect.
func TestAutoscalingGroupOSWithLaunchConfiguration(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()

	stubEC2DescribeImages(stub, "ami-0227c65b90645ae0c", "RunInstances:0002")
	stubCloudWatchASGQuery(stub, "deadbeef", 1) // don't care

	args := resources.AutoscalingGroup{
		LaunchConfiguration: &resources.LaunchConfiguration{AMI: "ami-0227c65b90645ae0c"},
	}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)
	assert.Equal(t, "windows", estimates.usage["operating_system"])
}

// Tests LaunchTemplate as a side effect.
func TestAutoscalingGroupOSWithLaunchTemplate(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()

	stubEC2DescribeImages(stub, "ami-0227c65b90645ae0c", "RunInstances:0002")
	stubCloudWatchASGQuery(stub, "deadbeef", 1) // don't care

	args := resources.AutoscalingGroup{
		LaunchTemplate: &resources.LaunchTemplate{AMI: "ami-0227c65b90645ae0c"},
	}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)
	assert.Equal(t, "windows", estimates.usage["operating_system"])
}

func TestAutoscalingGroupInstancesWithCloudWatch(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()

	stubEC2DescribeImages(stub, "ami-0227c65b90645ae0c", "RunInstances:0002") // don't care
	stubCloudWatchASGQuery(stub, "deadbeef", 3.14159)

	args := resources.AutoscalingGroup{
		Name: "deadbeef",
	}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)
	assert.Equal(t, int64(3), estimates.usage["instances"])
}

func TestAutoscalingGroupInstancesWithoutCloudWatch(t *testing.T) {
	stub := stubAWS(t)
	defer stub.Close()

	stubEC2DescribeImages(stub, "ami-0227c65b90645ae0c", "RunInstances:0002") // don't care
	stubCloudWatchASGQuery(stub, "deadbeef", 0)                               // no results
	stubEC2DescribeAutoscalingGroups(stub, "deadbeef", 5)

	args := resources.AutoscalingGroup{
		Name: "deadbeef",
	}
	resource := args.BuildResource()
	estimates := newEstimates(stub.ctx, t, resource)
	assert.Equal(t, int64(5), estimates.usage["instances"])
}
