package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAudioModelRatiosMatchOfficialCharacterPrices(t *testing.T) {
	tests := []struct {
		model     string
		cnyPer10K float64
	}{
		{model: "qwen3-tts-flash", cnyPer10K: 0.8},
		{model: "qwen3-tts-instruct-flash", cnyPer10K: 0.8},
		{model: "minimax-speech-2.8-turbo", cnyPer10K: 2},
		{model: "minimax-speech-2.8-hd", cnyPer10K: 3.5},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			ratio := GetDefaultModelRatioMap()[tt.model]
			assert.InDelta(t, tt.cnyPer10K*100/1000*RMB, ratio, 0.000000001)
			completionRatio, exists := defaultCompletionRatio[tt.model]
			assert.True(t, exists)
			assert.Equal(t, float64(0), completionRatio)
		})
	}
}
