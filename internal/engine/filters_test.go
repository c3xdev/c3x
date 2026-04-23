package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProductSelector(t *testing.T) {
	ps := &ProductSelector{
		VendorName:    strPtr("aws"),
		Service:       strPtr("AmazonEC2"),
		ProductFamily: strPtr("Compute Instance"),
		Region:        strPtr("us-east-1"),
	}

	assert.Equal(t, "aws", *ps.VendorName)
	assert.Equal(t, "AmazonEC2", *ps.Service)
	assert.Equal(t, "Compute Instance", *ps.ProductFamily)
	assert.Equal(t, "us-east-1", *ps.Region)
}

func TestRateSelector(t *testing.T) {
	rs := &RateSelector{
		PurchaseOption: strPtr("on_demand"),
		Unit:           strPtr("Hrs"),
	}

	assert.Equal(t, "on_demand", *rs.PurchaseOption)
	assert.Equal(t, "Hrs", *rs.Unit)
}

func TestAttributeMatch(t *testing.T) {
	// Exact value match
	am := &AttributeMatch{
		Key:   "instanceType",
		Value: strPtr("m5.xlarge"),
	}
	assert.Equal(t, "instanceType", am.Key)
	assert.Equal(t, "m5.xlarge", *am.Value)

	// Regex match
	regex := "/^m5\\..*$/"
	amRegex := &AttributeMatch{
		Key:        "instanceType",
		ValueRegex: &regex,
	}
	assert.NotNil(t, amRegex.ValueRegex)
}

func strPtr(s string) *string {
	return &s
}
