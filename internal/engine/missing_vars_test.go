package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestExtractMissingVarsCausingMissingAttributeKeys(t *testing.T) {
	spec := &ResourceSpec{
		Type:    "aws_instance",
		Address: "aws_instance.web",
		Metadata: map[string]gjson.Result{
			"attributesWithUnknownKeys": gjson.Parse(`[{"attribute":"tags","missingVariables":["var.env","var.name"]}]`),
		},
	}

	result := ExtractMissingVarsCausingMissingAttributeKeys(spec, "tags")
	assert.Len(t, result, 2)
	assert.Contains(t, result, "var.env")
	assert.Contains(t, result, "var.name")
}

func TestExtractMissingVars_NoMatch(t *testing.T) {
	spec := &ResourceSpec{
		Metadata: map[string]gjson.Result{
			"attributesWithUnknownKeys": gjson.Parse(`[{"attribute":"other","missingVariables":["var.x"]}]`),
		},
	}

	result := ExtractMissingVarsCausingMissingAttributeKeys(spec, "tags")
	assert.Empty(t, result)
}

func TestExtractMissingVars_NoMetadata(t *testing.T) {
	spec := &ResourceSpec{}
	result := ExtractMissingVarsCausingMissingAttributeKeys(spec, "tags")
	assert.Empty(t, result)
}
