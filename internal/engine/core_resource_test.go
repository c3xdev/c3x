package engine

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBlankCoreResource(t *testing.T) {
	blank := BlankCoreResource{Type: "aws_vpc"}

	assert.Equal(t, "aws_vpc", blank.CoreType())
	assert.Nil(t, blank.UsageSchema())

	blank.PopulateUsage(nil)

	r := blank.BuildResource()
	assert.True(t, r.NoPrice)
}

func TestNewPartialResource(t *testing.T) {
	spec := &ResourceSpec{Type: "aws_instance", Address: "aws_instance.web"}
	estimate := &Estimate{Name: "test"}
	var catalogItem CatalogItem = BlankCoreResource{Type: "aws_instance"}

	partial := NewPartialResource(spec, estimate, catalogItem, []string{"i-12345"})

	assert.Equal(t, "aws_instance", partial.Type)
	assert.Equal(t, "aws_instance.web", partial.Address)
	assert.NotNil(t, partial.CoreResource)
	assert.Equal(t, []string{"i-12345"}, partial.CloudResourceIDs)
}

func TestBuildResource_WithCoreResource(t *testing.T) {
	spec := &ResourceSpec{Type: "aws_instance", Address: "aws_instance.web"}
	catalogItem := BlankCoreResource{Type: "aws_instance"}
	partial := NewPartialResource(spec, nil, catalogItem, nil)

	result := BuildResource(partial, nil)

	assert.NotNil(t, result)
	assert.True(t, result.NoPrice)
}

func TestBuildResource_WithLegacyResource(t *testing.T) {
	spec := &ResourceSpec{Type: "aws_instance", Address: "aws_instance.web"}
	estimate := &Estimate{
		Name: "aws_instance.web",
		CostComponents: []*LineItem{
			{Name: "compute", MonthlyQuantity: dp(730), price: decimal.NewFromFloat(0.10)},
		},
	}
	partial := NewPartialResource(spec, estimate, nil, nil)

	result := BuildResource(partial, nil)

	assert.NotNil(t, result)
	assert.Equal(t, "aws_instance.web", result.Name)
}

func TestBuildEstimates(t *testing.T) {
	spec := &ResourceSpec{
		Type:    "aws_vpc",
		Address: "aws_vpc.main",
		RawValues: gjson.Parse("{}"),
	}
	catalogItem := BlankCoreResource{Type: "aws_vpc"}
	partial := NewPartialResource(spec, nil, catalogItem, nil)

	ws := &Workspace{
		Name:             "test-workspace",
		PartialResources: []*UnpricedEntry{partial},
	}

	usageMap := ConsumptionMap{}
	projectMap := map[*Workspace]ConsumptionMap{ws: usageMap}

	BuildEstimates([]*Workspace{ws}, projectMap)

	require.Len(t, ws.Resources, 1)
	assert.NotNil(t, ws.Resources[0])
	assert.True(t, ws.Resources[0].NoPrice)
}
