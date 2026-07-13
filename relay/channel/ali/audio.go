package ali

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const (
	aliAudioOutputFormatContextKey   = "ali_audio_output_format"
	aliAudioResponseFormatContextKey = "ali_audio_response_format"
)

type aliAudioMetadata struct {
	LanguageType         string   `json:"language_type,omitempty"`
	OptimizeInstructions *bool    `json:"optimize_instructions,omitempty"`
	SampleRate           *int     `json:"sample_rate,omitempty"`
	Bitrate              *int     `json:"bitrate,omitempty"`
	Channel              *int     `json:"channel,omitempty"`
	Volume               *float64 `json:"volume,omitempty"`
	Pitch                *int     `json:"pitch,omitempty"`
	Emotion              string   `json:"emotion,omitempty"`
	LanguageBoost        string   `json:"language_boost,omitempty"`
	TextNormalization    *bool    `json:"text_normalization,omitempty"`
	LatexRead            *bool    `json:"latex_read,omitempty"`
	OutputFormat         string   `json:"output_format,omitempty"`
}

type aliQwenAudioInput struct {
	Text                 string `json:"text"`
	Voice                string `json:"voice"`
	LanguageType         string `json:"language_type,omitempty"`
	Instructions         string `json:"instructions,omitempty"`
	OptimizeInstructions *bool  `json:"optimize_instructions,omitempty"`
}

type aliMiniMaxVoiceSetting struct {
	VoiceID           string   `json:"voice_id"`
	Speed             *float64 `json:"speed,omitempty"`
	Volume            *float64 `json:"vol,omitempty"`
	Pitch             *int     `json:"pitch,omitempty"`
	Emotion           string   `json:"emotion,omitempty"`
	TextNormalization *bool    `json:"text_normalization,omitempty"`
	LatexRead         *bool    `json:"latex_read,omitempty"`
}

type aliMiniMaxAudioSetting struct {
	SampleRate *int   `json:"sample_rate,omitempty"`
	Bitrate    *int   `json:"bitrate,omitempty"`
	Format     string `json:"format,omitempty"`
	Channel    *int   `json:"channel,omitempty"`
}

type aliMiniMaxAudioInput struct {
	Text          string                 `json:"text"`
	VoiceSetting  aliMiniMaxVoiceSetting `json:"voice_setting"`
	AudioSetting  aliMiniMaxAudioSetting `json:"audio_setting,omitempty"`
	LanguageBoost string                 `json:"language_boost,omitempty"`
	OutputFormat  string                 `json:"output_format,omitempty"`
}

type aliAudioRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"`
}

type aliAudioResponse struct {
	StatusCode int    `json:"status_code"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
	Output     struct {
		Audio struct {
			Data string `json:"data"`
			URL  string `json:"url"`
		} `json:"audio"`
		BaseResponse struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
		Data struct {
			Audio string `json:"audio"`
		} `json:"data"`
		ExtraInfo struct {
			UsageCharacters int    `json:"usage_characters"`
			AudioFormat     string `json:"audio_format"`
		} `json:"extra_info"`
	} `json:"output"`
	Usage struct {
		Characters int `json:"characters"`
	} `json:"usage"`
}

func convertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	if info.RelayMode != constant.RelayModeAudioSpeech {
		return nil, errors.New("unsupported audio relay mode")
	}

	metadata := aliAudioMetadata{}
	if len(request.Metadata) > 0 {
		if err := common.Unmarshal(request.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("invalid Ali audio metadata: %w", err)
		}
	}

	model := request.Model
	if info.ChannelMeta != nil && info.UpstreamModelName != "" {
		model = info.UpstreamModelName
	}
	converted := aliAudioRequest{Model: model}
	switch model {
	case "qwen3-tts-flash", "qwen3-tts-instruct-flash":
		input := aliQwenAudioInput{
			Text:         request.Input,
			Voice:        request.Voice,
			LanguageType: metadata.LanguageType,
		}
		if model == "qwen3-tts-instruct-flash" {
			input.Instructions = request.Instructions
			if request.Instructions != "" {
				input.OptimizeInstructions = metadata.OptimizeInstructions
			}
		}
		converted.Input = input
	case "MiniMax/speech-2.8-turbo", "MiniMax/speech-2.8-hd":
		responseFormat := request.ResponseFormat
		if responseFormat == "" {
			responseFormat = "mp3"
		}
		outputFormat := metadata.OutputFormat
		if outputFormat == "" {
			outputFormat = "hex"
		}
		if outputFormat != "hex" && outputFormat != "url" {
			return nil, errors.New("Ali MiniMax output_format must be hex or url")
		}
		converted.Input = aliMiniMaxAudioInput{
			Text: request.Input,
			VoiceSetting: aliMiniMaxVoiceSetting{
				VoiceID:           request.Voice,
				Speed:             request.Speed,
				Volume:            metadata.Volume,
				Pitch:             metadata.Pitch,
				Emotion:           metadata.Emotion,
				TextNormalization: metadata.TextNormalization,
				LatexRead:         metadata.LatexRead,
			},
			AudioSetting: aliMiniMaxAudioSetting{
				SampleRate: metadata.SampleRate,
				Bitrate:    metadata.Bitrate,
				Format:     responseFormat,
				Channel:    metadata.Channel,
			},
			LanguageBoost: metadata.LanguageBoost,
			OutputFormat:  outputFormat,
		}
		c.Set(aliAudioOutputFormatContextKey, outputFormat)
		c.Set(aliAudioResponseFormatContextKey, responseFormat)
	default:
		return nil, fmt.Errorf("unsupported Ali audio model: %s", model)
	}

	data, err := common.Marshal(converted)
	if err != nil {
		return nil, fmt.Errorf("marshal Ali audio request: %w", err)
	}
	return bytes.NewReader(data), nil
}

func handleAudioResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (any, *types.NewAPIError) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewErrorWithStatusCode(err, types.ErrorCodeReadResponseBodyFailed, http.StatusBadGateway)
	}

	response := aliAudioResponse{}
	if err = common.Unmarshal(body, &response); err != nil {
		return nil, types.NewErrorWithStatusCode(errors.New("failed to parse Ali audio response"), types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}
	if response.StatusCode != 0 && response.StatusCode != http.StatusOK {
		return nil, aliAudioUpstreamError(info, response.Code, response.Message, response.StatusCode)
	}
	if response.Output.BaseResponse.StatusCode != 0 {
		return nil, aliAudioUpstreamError(info, fmt.Sprintf("%d", response.Output.BaseResponse.StatusCode), response.Output.BaseResponse.StatusMsg, http.StatusBadRequest)
	}

	characters := response.Usage.Characters
	if characters == 0 {
		characters = response.Output.ExtraInfo.UsageCharacters
	}
	usage := &dto.Usage{PromptTokens: characters, TotalTokens: characters}
	usage.PromptTokensDetails.TextTokens = characters

	if response.Output.Audio.URL != "" {
		c.Header("Location", response.Output.Audio.URL)
		c.Status(http.StatusFound)
		c.Writer.WriteHeaderNow()
		return usage, nil
	}
	if response.Output.Audio.Data != "" {
		data, decodeErr := base64.StdEncoding.DecodeString(response.Output.Audio.Data)
		if decodeErr != nil {
			return nil, types.NewErrorWithStatusCode(errors.New("failed to decode Ali Qwen audio data"), types.ErrorCodeBadResponseBody, http.StatusBadGateway)
		}
		format := c.GetString(aliAudioResponseFormatContextKey)
		c.Data(http.StatusOK, aliAudioContentType(format), data)
		return usage, nil
	}
	if response.Output.Data.Audio != "" {
		if c.GetString(aliAudioOutputFormatContextKey) == "url" || strings.HasPrefix(response.Output.Data.Audio, "http://") || strings.HasPrefix(response.Output.Data.Audio, "https://") {
			c.Header("Location", response.Output.Data.Audio)
			c.Status(http.StatusFound)
			c.Writer.WriteHeaderNow()
			return usage, nil
		}
		data, decodeErr := hex.DecodeString(response.Output.Data.Audio)
		if decodeErr != nil {
			return nil, types.NewErrorWithStatusCode(errors.New("failed to decode Ali MiniMax audio data"), types.ErrorCodeBadResponseBody, http.StatusBadGateway)
		}
		format := c.GetString(aliAudioResponseFormatContextKey)
		if format == "" {
			format = response.Output.ExtraInfo.AudioFormat
		}
		c.Data(http.StatusOK, aliAudioContentType(format), data)
		return usage, nil
	}

	return nil, types.NewErrorWithStatusCode(errors.New("Ali audio response did not contain audio"), types.ErrorCodeEmptyResponse, http.StatusBadGateway)
}

func aliAudioUpstreamError(info *relaycommon.RelayInfo, code, message string, status int) *types.NewAPIError {
	if info != nil && info.ChannelMeta != nil && info.ApiKey != "" {
		code = strings.ReplaceAll(code, info.ApiKey, "[REDACTED]")
		message = strings.ReplaceAll(message, info.ApiKey, "[REDACTED]")
	}
	if code == "" {
		code = "upstream_error"
	}
	return types.NewErrorWithStatusCode(fmt.Errorf("Ali audio error %s: %s", code, message), types.ErrorCodeBadResponse, status)
}

func aliAudioContentType(format string) string {
	switch strings.ToLower(format) {
	case "wav":
		return "audio/wav"
	case "flac":
		return "audio/flac"
	case "pcm":
		return "audio/pcm"
	default:
		return "audio/mpeg"
	}
}
