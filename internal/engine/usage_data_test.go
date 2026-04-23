package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestNewUsageData(t *testing.T) {
	attrs := map[string]gjson.Result{
		"monthly_requests": gjson.Parse("1000"),
	}

	ud := NewUsageData("aws_lambda.test", attrs)

	assert.Equal(t, "aws_lambda.test", ud.Address)
	assert.Equal(t, int64(1000), ud.Get("monthly_requests").Int())
}

func TestConsumptionProfile_Get(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{
		"count": gjson.Parse("42"),
	})

	result := ud.Get("count")
	assert.Equal(t, int64(42), result.Int())

	missing := ud.Get("nonexistent")
	assert.False(t, missing.Exists())
}

func TestConsumptionProfile_GetFloat(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{
		"rate": gjson.Parse("3.14"),
	})

	result := ud.GetFloat("rate")
	assert.NotNil(t, result)
	assert.InDelta(t, 3.14, *result, 0.001)

	missing := ud.GetFloat("nope")
	assert.Nil(t, missing)
}

func TestConsumptionProfile_GetInt(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{
		"count": gjson.Parse("42"),
	})

	result := ud.GetInt("count")
	assert.NotNil(t, result)
	assert.Equal(t, int64(42), *result)
}

func TestConsumptionProfile_GetString(t *testing.T) {
	ud := NewUsageData("test", map[string]gjson.Result{
		"name": gjson.Parse(`"hello"`),
	})

	result := ud.GetString("name")
	assert.NotNil(t, result)
	assert.Equal(t, "hello", *result)
}

func TestConsumptionProfile_Copy(t *testing.T) {
	original := NewUsageData("test", map[string]gjson.Result{
		"key1": gjson.Parse("100"),
	})

	copied := original.Copy()
	assert.Equal(t, original.Address, copied.Address)
}

func TestParseAttributes(t *testing.T) {
	input := map[string]interface{}{
		"string_val": "hello",
		"int_val":    42,
	}

	result := ParseAttributes(input)
	assert.Contains(t, result, "string_val")
	assert.Contains(t, result, "int_val")
}
