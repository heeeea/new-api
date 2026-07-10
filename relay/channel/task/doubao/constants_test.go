package doubao

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVideoInputRatioSupportsSeedance20Mini(t *testing.T) {
	ratio, ok := GetVideoInputRatio("doubao-seedance-2-0-mini-260615", "720p", true)

	require.True(t, ok)
	require.InDelta(t, 14.0/23.0, ratio, 0.000001)
}

func TestModelListIncludesSeedance20Mini(t *testing.T) {
	assert.Contains(t, ModelList, "doubao-seedance-2-0-mini-260615")
}
