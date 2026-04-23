package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestNewResourceSpec(t *testing.T) {
	tags := map[string]string{"env": "prod"}
	raw := gjson.Parse(`{"instance_type": "t3.micro", "ami": "ami-123"}`)

	spec := NewResourceData("aws_instance", "aws", "aws_instance.web", &tags, raw)

	assert.Equal(t, "aws_instance", spec.Type)
	assert.Equal(t, "aws", spec.ProviderName)
	assert.Equal(t, "aws_instance.web", spec.Address)
	assert.Equal(t, "prod", (*spec.Tags)["env"])
}

func TestResourceSpec_Get(t *testing.T) {
	raw := gjson.Parse(`{"instance_type": "t3.micro", "count": 3, "nested": {"key": "val"}}`)
	spec := NewResourceData("aws_instance", "aws", "test", nil, raw)

	assert.Equal(t, "t3.micro", spec.Get("instance_type").String())
	assert.Equal(t, int64(3), spec.Get("count").Int())
	assert.Equal(t, "val", spec.Get("nested.key").String())
	assert.False(t, spec.Get("nonexistent").Exists())
}

func TestResourceSpec_GetStringOrDefault(t *testing.T) {
	raw := gjson.Parse(`{"region": "us-east-1"}`)
	spec := NewResourceData("test", "aws", "test", nil, raw)

	assert.Equal(t, "us-east-1", spec.GetStringOrDefault("region", "us-west-2"))
	assert.Equal(t, "us-west-2", spec.GetStringOrDefault("missing", "us-west-2"))
}

func TestResourceSpec_GetInt64OrDefault(t *testing.T) {
	raw := gjson.Parse(`{"count": 5}`)
	spec := NewResourceData("test", "aws", "test", nil, raw)

	assert.Equal(t, int64(5), spec.GetInt64OrDefault("count", 1))
	assert.Equal(t, int64(1), spec.GetInt64OrDefault("missing", 1))
}

func TestResourceSpec_IsEmpty(t *testing.T) {
	empty := NewResourceData("test", "aws", "test", nil, gjson.Parse("{}"))
	assert.True(t, empty.IsEmpty("key"))

	nonEmpty := NewResourceData("test", "aws", "test", nil, gjson.Parse(`{"key": "value"}`))
	assert.False(t, nonEmpty.IsEmpty("key"))
}

func TestResourceSpec_References(t *testing.T) {
	spec := NewResourceData("test", "aws", "test", nil, gjson.Parse("{}"))
	ref := NewResourceData("aws_vpc", "aws", "aws_vpc.main", nil, gjson.Parse("{}"))

	spec.AddReference("vpc_id", ref, nil)

	refs := spec.References("vpc_id")
	assert.Len(t, refs, 1)
	assert.Equal(t, "aws_vpc.main", refs[0].Address)
}

func TestAddRawValue(t *testing.T) {
	original := gjson.Parse(`{"key1": "val1"}`)

	updated := AddRawValue(original, "key2", "val2")

	assert.Equal(t, "val1", updated.Get("key1").String())
	assert.Equal(t, "val2", updated.Get("key2").String())
}
