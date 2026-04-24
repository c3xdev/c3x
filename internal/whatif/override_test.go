package whatif

import (
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestParseOverride_Valid(t *testing.T) {
	o, err := ParseOverride("aws_instance.web.instance_type=m6i.xlarge")
	require.NoError(t, err)
	assert.Equal(t, "aws_instance.web", o.ResourceAddress)
	assert.Equal(t, "instance_type", o.AttributePath)
	assert.Equal(t, "m6i.xlarge", o.Value)
}

func TestParseOverride_BoolValue(t *testing.T) {
	o, err := ParseOverride("aws_db_instance.main.multi_az=true")
	require.NoError(t, err)
	assert.Equal(t, "aws_db_instance.main", o.ResourceAddress)
	assert.Equal(t, "multi_az", o.AttributePath)
	assert.Equal(t, "true", o.Value)
}

func TestParseOverride_NoEquals(t *testing.T) {
	_, err := ParseOverride("aws_instance.web.instance_type")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected 'resource_type.name.attribute=value'")
}

func TestParseOverride_TooFewParts(t *testing.T) {
	_, err := ParseOverride("aws_instance=m6i.xlarge")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected 'resource_type.name.attribute'")
}

func TestParseOverride_ValueWithEquals(t *testing.T) {
	o, err := ParseOverride("aws_instance.web.user_data=key=value")
	require.NoError(t, err)
	assert.Equal(t, "aws_instance.web", o.ResourceAddress)
	assert.Equal(t, "user_data", o.AttributePath)
	assert.Equal(t, "key=value", o.Value)
}

func TestMatchesAddress_Exact(t *testing.T) {
	assert.True(t, matchesAddress("aws_instance.web", "aws_instance.web"))
}

func TestMatchesAddress_WithModule(t *testing.T) {
	assert.True(t, matchesAddress("module.vpc.aws_instance.web", "aws_instance.web"))
}

func TestMatchesAddress_NoMatch(t *testing.T) {
	assert.False(t, matchesAddress("aws_instance.api", "aws_instance.web"))
}

func TestApplyOverrides(t *testing.T) {
	projects := []*engine.Workspace{
		{
			PartialResources: []*engine.UnpricedEntry{
				{
					Address: "aws_instance.web",
					Metadata: map[string]gjson.Result{
						"values": gjson.Parse(`{"instance_type":"t3.micro","ami":"ami-123"}`),
					},
				},
			},
		},
	}

	overrides := []*Override{
		{ResourceAddress: "aws_instance.web", AttributePath: "instance_type", Value: "m6i.xlarge"},
	}

	applied, err := ApplyOverrides(projects, overrides)
	require.NoError(t, err)
	assert.Equal(t, 1, applied)

	// Verify the metadata was modified
	values := projects[0].PartialResources[0].Metadata["values"]
	assert.Equal(t, "m6i.xlarge", values.Get("instance_type").String())
	// Original attributes preserved
	assert.Equal(t, "ami-123", values.Get("ami").String())
}

func TestApplyOverrides_NoMatch(t *testing.T) {
	projects := []*engine.Workspace{
		{
			PartialResources: []*engine.UnpricedEntry{
				{
					Address: "aws_instance.api",
					Metadata: map[string]gjson.Result{
						"values": gjson.Parse(`{"instance_type":"t3.micro"}`),
					},
				},
			},
		},
	}

	overrides := []*Override{
		{ResourceAddress: "aws_instance.web", AttributePath: "instance_type", Value: "m6i.xlarge"},
	}

	applied, err := ApplyOverrides(projects, overrides)
	require.NoError(t, err)
	assert.Equal(t, 0, applied)
}

func TestSetJSONValue(t *testing.T) {
	result := setJSONValue(`{"a":"1","b":"2"}`, "a", "changed")
	parsed := gjson.Parse(result)
	assert.Equal(t, "changed", parsed.Get("a").String())
	assert.Equal(t, "2", parsed.Get("b").String())
}

func TestSetJSONValue_NewKey(t *testing.T) {
	result := setJSONValue(`{"a":"1"}`, "b", "new")
	parsed := gjson.Parse(result)
	assert.Equal(t, "1", parsed.Get("a").String())
	assert.Equal(t, "new", parsed.Get("b").String())
}

func TestSetJSONValue_EmptyJSON(t *testing.T) {
	result := setJSONValue(``, "key", "value")
	parsed := gjson.Parse(result)
	assert.Equal(t, "value", parsed.Get("key").String())
}

func TestSetJSONValue_NonObjectJSON_PreservesOriginal(t *testing.T) {
	// Arrays, numbers, and malformed JSON should be preserved, not wiped.
	original := `[1,2,3]`
	result := setJSONValue(original, "key", "value")
	assert.Equal(t, original, result)
}

func TestParseOverride_ModuleAddress(t *testing.T) {
	o, err := ParseOverride("module.vpc.aws_instance.web.instance_type=m5.large")
	assert.NoError(t, err)
	assert.Equal(t, "module.vpc.aws_instance.web", o.ResourceAddress)
	assert.Equal(t, "instance_type", o.AttributePath)
	assert.Equal(t, "m5.large", o.Value)
}
