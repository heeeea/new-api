package gemini

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVeoResolutionRatioMatchesOfficialGeminiAPIPrices(t *testing.T) {
	tests := []struct {
		model      string
		resolution string
		want       float64
	}{
		{"veo-3.1-fast-generate-preview", "720p", 1},
		{"veo-3.1-fast-generate-preview", "1080p", 1.2},
		{"veo-3.1-fast-generate-preview", "4k", 3},
		{"veo-3.1-generate-preview", "720p", 1},
		{"veo-3.1-generate-preview", "1080p", 1},
		{"veo-3.1-generate-preview", "4k", 1.5},
	}
	for _, test := range tests {
		t.Run(test.model+"/"+test.resolution, func(t *testing.T) {
			require.InDelta(t, test.want, VeoResolutionRatio(test.model, test.resolution), 0.000001)
		})
	}
}
