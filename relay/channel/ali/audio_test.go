package ali

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertAudioRequestQwenUsesOnlyOfficialControls(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := aliAudioTestInfo("qwen3-tts-instruct-flash")

	body, err := (&Adaptor{}).ConvertAudioRequest(c, info, dto.AudioRequest{
		Model:          "qwen3-tts-instruct-flash",
		Input:          "新品限时开售",
		Voice:          "Cherry",
		Instructions:   "语速明快，像电商主播",
		ResponseFormat: "mp3",
		Speed:          common.GetPointer(1.25),
		Metadata:       json.RawMessage(`{"language_type":"Chinese","optimize_instructions":true,"sample_rate":44100,"emotion":"happy"}`),
	})
	require.NoError(t, err)

	var request map[string]any
	require.NoError(t, common.DecodeJson(body, &request))
	assert.Equal(t, "qwen3-tts-instruct-flash", request["model"])
	assert.Equal(t, map[string]any{
		"text":                  "新品限时开售",
		"voice":                 "Cherry",
		"language_type":         "Chinese",
		"instructions":          "语速明快，像电商主播",
		"optimize_instructions": true,
	}, request["input"])
}

func TestConvertAudioRequestQwenFlashOmitsInstructions(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := aliAudioTestInfo("qwen3-tts-flash")

	body, err := (&Adaptor{}).ConvertAudioRequest(c, info, dto.AudioRequest{
		Model:        "qwen3-tts-flash",
		Input:        "新品限时开售",
		Voice:        "Cherry",
		Instructions: "这个模型不应收到该字段",
	})
	require.NoError(t, err)

	var request struct {
		Input map[string]any `json:"input"`
	}
	require.NoError(t, common.DecodeJson(body, &request))
	assert.NotContains(t, request.Input, "instructions")
	assert.NotContains(t, request.Input, "optimize_instructions")
}

func TestConvertAudioRequestMiniMaxMapsOfficialControls(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := aliAudioTestInfo("MiniMax/speech-2.8-turbo")

	body, err := (&Adaptor{}).ConvertAudioRequest(c, info, dto.AudioRequest{
		Model:          "MiniMax/speech-2.8-turbo",
		Input:          "新品限时开售",
		Voice:          "male-qn-qingse",
		ResponseFormat: "wav",
		Speed:          common.GetPointer(1.2),
		Metadata: json.RawMessage(`{
			"sample_rate":24000,
			"bitrate":128000,
			"channel":1,
			"volume":1.5,
			"pitch":2,
			"emotion":"happy",
			"language_boost":"Chinese",
			"text_normalization":true,
			"latex_read":false,
			"output_format":"url"
		}`),
	})
	require.NoError(t, err)

	var request map[string]any
	require.NoError(t, common.DecodeJson(body, &request))
	assert.Equal(t, "MiniMax/speech-2.8-turbo", request["model"])
	assert.Equal(t, map[string]any{
		"text": "新品限时开售",
		"voice_setting": map[string]any{
			"voice_id":           "male-qn-qingse",
			"speed":              1.2,
			"vol":                1.5,
			"pitch":              float64(2),
			"emotion":            "happy",
			"text_normalization": true,
			"latex_read":         false,
		},
		"audio_setting": map[string]any{
			"sample_rate": float64(24000),
			"bitrate":     float64(128000),
			"format":      "wav",
			"channel":     float64(1),
		},
		"language_boost": "Chinese",
		"output_format":  "url",
	}, request["input"])
	assert.Equal(t, "url", c.GetString("ali_audio_output_format"))
}

func TestAliAudioRequestURLUsesMultimodalGenerationEndpoint(t *testing.T) {
	info := aliAudioTestInfo("qwen3-tts-flash")
	info.ChannelBaseUrl = "https://dashscope.aliyuncs.com/api/v1"
	url, err := (&Adaptor{}).GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation", url)
}

func TestAliAudioResponseNormalizesQwenURLAndMiniMaxHex(t *testing.T) {
	t.Run("qwen url", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", nil)
		usage, apiErr := (&Adaptor{}).DoResponse(c, &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"status_code":200,
				"output":{"audio":{"url":"https://cdn.example.test/audio.wav"}},
				"usage":{"characters":6}
			}`)),
		}, aliAudioTestInfo("qwen3-tts-flash"))
		require.Nil(t, apiErr)
		assert.Equal(t, http.StatusFound, recorder.Code)
		assert.Equal(t, "https://cdn.example.test/audio.wav", recorder.Header().Get("Location"))
		assert.Equal(t, 6, usage.(*dto.Usage).TotalTokens)
	})

	t.Run("minimax hex", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", nil)
		c.Set("ali_audio_response_format", "wav")
		usage, apiErr := (&Adaptor{}).DoResponse(c, &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"output":{
					"base_resp":{"status_code":0,"status_msg":"success"},
					"data":{"audio":"52494646","status":2},
					"extra_info":{"usage_characters":4}
				},
				"usage":{"characters":4}
			}`)),
		}, aliAudioTestInfo("MiniMax/speech-2.8-hd"))
		require.Nil(t, apiErr)
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "audio/wav", recorder.Header().Get("Content-Type"))
		assert.Equal(t, []byte("RIFF"), recorder.Body.Bytes())
		assert.Equal(t, 4, usage.(*dto.Usage).TotalTokens)
	})
}

func TestAliAudioResponseRedactsCredentialFromUpstreamError(t *testing.T) {
	const apiKey = "sk-test-secret-value"
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", nil)
	info := aliAudioTestInfo("qwen3-tts-flash")
	info.ApiKey = apiKey
	_, apiErr := (&Adaptor{}).DoResponse(c, &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body: io.NopCloser(strings.NewReader(`{
			"status_code":401,
			"code":"InvalidApiKey-sk-test-secret-value",
			"message":"Authorization Bearer sk-test-secret-value is invalid"
		}`)),
	}, info)
	require.NotNil(t, apiErr)
	assert.NotContains(t, apiErr.Error(), apiKey)
	assert.Contains(t, apiErr.Error(), "InvalidApiKey")
}

func aliAudioTestInfo(model string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		RelayMode: constant.RelayModeAudioSpeech,
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: model,
		},
	}
}
