package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsageEstimator_Type(t *testing.T) {
	// UsageEstimator is a function type
	var fn UsageEstimator
	assert.Nil(t, fn) // zero value is nil
}

func TestRemediater_Interface(t *testing.T) {
	// Remediater is an interface — verify it exists as a type
	var r Remediater
	assert.Nil(t, r)
}
