package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPolicies_Len(t *testing.T) {
	policies := Policies{{}, {}, {}}
	assert.Equal(t, 3, policies.Len())
}

func TestPolicy_Fields(t *testing.T) {
	p := Policy{
		ID:           "pol-123",
		Title:        "Max Monthly Cost",
		ResourceType: "aws_instance",
		Address:      "aws_instance.web",
	}
	assert.Equal(t, "pol-123", p.ID)
	assert.Equal(t, "aws_instance", p.ResourceType)
}
