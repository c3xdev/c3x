package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerraformSourceMap_Decode(t *testing.T) {
	tsm := TerraformSourceMap{}
	err := tsm.Decode("git::https://example.com=/local/path")
	assert.NoError(t, err)
}
