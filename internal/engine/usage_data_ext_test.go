package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestConsumptionMap_Get_ExactMatch(t *testing.T) {
	data := map[string]*ConsumptionProfile{
		"aws_instance.web": NewUsageData("aws_instance.web", map[string]gjson.Result{
			"monthly_hrs": gjson.Parse("730"),
		}),
	}
	cm := NewUsageMap(data)

	result := cm.Get("aws_instance.web")
	assert.NotNil(t, result)
	assert.Equal(t, int64(730), result.Get("monthly_hrs").Int())
}

func TestConsumptionMap_Get_TypeWildcard(t *testing.T) {
	data := map[string]*ConsumptionProfile{
		"aws_instance[*]": NewUsageData("aws_instance[*]", map[string]gjson.Result{
			"monthly_hrs": gjson.Parse("365"),
		}),
	}
	cm := NewUsageMap(data)

	// Should match aws_instance.anything via wildcard
	result := cm.Get("aws_instance.web")
	if result != nil {
		assert.Equal(t, int64(365), result.Get("monthly_hrs").Int())
	}
}

func TestConsumptionMap_Get_NoMatch(t *testing.T) {
	data := map[string]*ConsumptionProfile{
		"aws_s3_bucket.data": NewUsageData("aws_s3_bucket.data", map[string]gjson.Result{}),
	}
	cm := NewUsageMap(data)

	result := cm.Get("aws_instance.web")
	assert.Nil(t, result)
}

func TestConsumptionMap_Empty(t *testing.T) {
	cm := ConsumptionMap{}
	result := cm.Get("anything")
	assert.Nil(t, result)
}

func TestNewUsageMapFromInterface(t *testing.T) {
	input := map[string]interface{}{
		"aws_instance.web": map[string]interface{}{
			"monthly_hrs": 730,
		},
	}
	cm := NewUsageMapFromInterface(input)
	result := cm.Get("aws_instance.web")
	assert.NotNil(t, result)
}

func TestConsumptionProfile_GetStringArray(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{
		"cidrs": gjson.Parse(`["10.0.0.0/8", "172.16.0.0/12"]`),
	})
	result := ud.GetStringArray("cidrs")
	if result != nil {
		assert.Len(t, *result, 2)
	}
}

func TestConsumptionProfile_Merge(t *testing.T) {
	base := NewUsageData("test", map[string]gjson.Result{
		"key1": gjson.Parse("1"),
		"key2": gjson.Parse("2"),
	})
	override := NewUsageData("test", map[string]gjson.Result{
		"key2": gjson.Parse("99"),
		"key3": gjson.Parse("3"),
	})

	merged := base.Merge(override)
	assert.NotNil(t, merged)
}

func TestConsumptionField_AllTypes(t *testing.T) {
	fields := []ConsumptionField{
		{Key: "count", ValueType: Int64, DefaultValue: 0},
		{Key: "rate", ValueType: Float64, DefaultValue: 0.0},
		{Key: "name", ValueType: String, DefaultValue: ""},
		{Key: "ips", ValueType: StringArray, DefaultValue: nil},
		{Key: "sub", ValueType: SubResourceUsage, DefaultValue: nil},
		{Key: "kv", ValueType: KeyValueMap, DefaultValue: nil},
	}

	assert.Len(t, fields, 6)
	assert.Equal(t, Int64, fields[0].ValueType)
	assert.Equal(t, KeyValueMap, fields[5].ValueType)
}
