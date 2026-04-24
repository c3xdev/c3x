package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsumptionField_Types(t *testing.T) {
	tests := []struct {
		name      string
		valueType UsageVariableType
	}{
		{"Int64", Int64},
		{"Float64", Float64},
		{"String", String},
		{"StringArray", StringArray},
		{"SubResourceUsage", SubResourceUsage},
		{"KeyValueMap", KeyValueMap},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := ConsumptionField{
				Key:          "test_field",
				DefaultValue: 0,
				ValueType:    tt.valueType,
				Description:  "Test field",
			}

			assert.Equal(t, "test_field", field.Key)
			assert.Equal(t, tt.valueType, field.ValueType)
		})
	}
}

func TestConsumptionField_DefaultValues(t *testing.T) {
	intField := ConsumptionField{Key: "count", DefaultValue: int64(0), ValueType: Int64}
	assert.Equal(t, int64(0), intField.DefaultValue)

	floatField := ConsumptionField{Key: "rate", DefaultValue: 0.0, ValueType: Float64}
	assert.Equal(t, 0.0, floatField.DefaultValue)

	strField := ConsumptionField{Key: "name", DefaultValue: "", ValueType: String}
	assert.Equal(t, "", strField.DefaultValue)
}
