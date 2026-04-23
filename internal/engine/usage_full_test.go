package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestConsumptionMap_TypeDefaults(t *testing.T) {
	// Type defaults use the pattern "resource_type[*]"
	data := map[string]*ConsumptionProfile{
		"aws_instance[*]": NewUsageData("aws_instance[*]", map[string]gjson.Result{
			"monthly_hrs": gjson.Parse("365"),
		}),
	}
	cm := NewUsageMap(data)

	// Should match any aws_instance
	result := cm.Get("aws_instance.web")
	if result != nil {
		assert.Equal(t, int64(365), result.Get("monthly_hrs").Int())
	}
}

func TestConsumptionMap_Precedence(t *testing.T) {
	// Exact match should take precedence over wildcard
	data := map[string]*ConsumptionProfile{
		"aws_instance.web":  NewUsageData("aws_instance.web", map[string]gjson.Result{"hours": gjson.Parse("730")}),
		"aws_instance[*]":   NewUsageData("aws_instance[*]", map[string]gjson.Result{"hours": gjson.Parse("365")}),
	}
	cm := NewUsageMap(data)

	exact := cm.Get("aws_instance.web")
	if exact != nil {
		assert.Equal(t, int64(730), exact.Get("hours").Int())
	}
}



func TestConsumptionProfile_GetFloat_Missing(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{})
	result := ud.GetFloat("missing")
	assert.Nil(t, result)
}

func TestConsumptionProfile_GetInt_Missing(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{})
	result := ud.GetInt("missing")
	assert.Nil(t, result)
}

func TestConsumptionProfile_GetString_Missing(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{})
	result := ud.GetString("missing")
	assert.Nil(t, result)
}

func TestConsumptionProfile_GetStringArray_Missing(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{})
	result := ud.GetStringArray("missing")
	assert.Nil(t, result)
}

func TestConsumptionProfile_GetStringArray_Present(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{
		"ips": gjson.Parse(`["10.0.0.1", "10.0.0.2"]`),
	})
	result := ud.GetStringArray("ips")
	assert.NotNil(t, result)
	assert.Len(t, *result, 2)
}

func TestConsumptionProfile_Copy_Nil(t *testing.T) {
	var ud *ConsumptionProfile
	copied := ud.Copy()
	assert.Nil(t, copied)
}

func TestConsumptionProfile_Merge_Nil(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{"k": gjson.Parse("1")})
	result := ud.Merge(nil)
	assert.NotNil(t, result)
}

func TestNewUsageMap_Nil(t *testing.T) {
	cm := NewUsageMap(nil)
	result := cm.Get("anything")
	assert.Nil(t, result)
}

func TestParseAttributes_Nil(t *testing.T) {
	result := ParseAttributes(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestParseAttributes_Complex(t *testing.T) {
	input := map[string]interface{}{
		"string":  "hello",
		"int":     42,
		"float":   3.14,
		"bool":    true,
		"nested":  map[string]interface{}{"key": "val"},
		"list":    []interface{}{1, 2, 3},
	}
	result := ParseAttributes(input)
	assert.NotEmpty(t, result)
}
