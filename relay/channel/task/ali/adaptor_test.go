package ali

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func testRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
}

func TestConvertToAliRequestWan27I2VBuildsMediaFromImage(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "wan2.7-i2v",
		Prompt:   "animate the first frame",
		Image:    "https://example.com/first.png",
		Size:     "720p",
		Duration: 10,
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, "wan2.7-i2v", aliReq.Model)
	require.Equal(t, "720P", aliReq.Parameters.Resolution)
	require.Equal(t, 10, aliReq.Parameters.Duration)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_frame", URL: "https://example.com/first.png"},
	}, aliReq.Input.Media)
	require.Empty(t, aliReq.Input.ImgURL)

	body, err := common.Marshal(aliReq)
	require.NoError(t, err)
	require.Contains(t, string(body), `"media"`)
	require.NotContains(t, string(body), `"img_url"`)
}

func TestConvertToAliRequestWan27I2VBuildsFirstAndLastFrameFromImages(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.7-i2v",
		Prompt: "interpolate between frames",
		Images: []string{
			"https://example.com/first.png",
			"https://example.com/last.png",
		},
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_frame", URL: "https://example.com/first.png"},
		{Type: "last_frame", URL: "https://example.com/last.png"},
	}, aliReq.Input.Media)
}

func TestConvertToAliRequestWan27I2VPrefersImageBeforeImagesAndInputReference(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:          "wan2.7-i2v",
		Prompt:         "use the direct image",
		Image:          " https://example.com/direct.png ",
		Images:         []string{"https://example.com/images-first.png", " https://example.com/images-last.png "},
		InputReference: "https://example.com/input-reference.png",
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_frame", URL: "https://example.com/direct.png"},
		{Type: "last_frame", URL: "https://example.com/images-last.png"},
	}, aliReq.Input.Media)
}

func TestConvertToAliRequestWan27I2VFallsBackToFirstNonEmptyImage(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.7-i2v",
		Prompt: "skip blank images",
		Image:  " ",
		Images: []string{
			" ",
			" https://example.com/first.png ",
			" https://example.com/last.png ",
		},
		InputReference: "https://example.com/input-reference.png",
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_frame", URL: "https://example.com/first.png"},
		{Type: "last_frame", URL: "https://example.com/last.png"},
	}, aliReq.Input.Media)
}

func TestConvertToAliRequestWan27I2VKeepsExplicitMetadataMedia(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:          "wan2.7-i2v",
		Prompt:         "continue the clip",
		Image:          "https://example.com/direct.png",
		Images:         []string{"https://example.com/images-first.png", "https://example.com/images-last.png"},
		InputReference: "https://example.com/input-reference.png",
		Metadata: map[string]interface{}{
			"input": map[string]interface{}{
				"media": []interface{}{
					map[string]interface{}{
						"type": "first_clip",
						"url":  "https://example.com/input.mp4",
					},
				},
			},
		},
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, []AliVideoMedia{
		{Type: "first_clip", URL: "https://example.com/input.mp4"},
	}, aliReq.Input.Media)
	require.Empty(t, aliReq.Input.ImgURL)

	body, err := common.Marshal(aliReq)
	require.NoError(t, err)
	require.Contains(t, string(body), `"media"`)
	require.NotContains(t, string(body), `"img_url"`)
}

func TestConvertToAliRequestWan27I2VRequiresMedia(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.7-i2v",
		Prompt: "animate without a frame",
	}

	_, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "requires image"))
}

func TestConvertToAliRequestWan25I2VKeepsLegacyImgURL(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.5-i2v-preview",
		Prompt: "animate the first frame",
		Image:  "https://example.com/first.png",
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, "https://example.com/first.png", aliReq.Input.ImgURL)
	require.Empty(t, aliReq.Input.Media)

	body, err := common.Marshal(aliReq)
	require.NoError(t, err)
	require.Contains(t, string(body), `"img_url"`)
	require.NotContains(t, string(body), `"media"`)
}

func TestConvertToAliRequestHappyHorseUsesReferenceMedia(t *testing.T) {
	adaptor := &TaskAdaptor{}
	info := testRelayInfo()
	info.OriginModelName = "happy-horse-1.1"
	info.IsModelMapped = true
	info.UpstreamModelName = "happyhorse-1.1-r2v"
	req := relaycommon.TaskSubmitReq{
		Model:    "happy-horse-1.1",
		Prompt:   "show the product from several angles",
		Images:   []string{"https://example.com/product.png", "https://example.com/detail.png"},
		Duration: 5,
		Size:     "720p",
		Metadata: map[string]interface{}{"ratio": "9:16"},
	}

	aliReq, err := adaptor.convertToAliRequest(info, req)

	require.NoError(t, err)
	require.Equal(t, "happyhorse-1.1-r2v", aliReq.Model)
	require.Equal(t, "9:16", aliReq.Parameters.Ratio)
	require.Contains(t, aliReq.Input.Prompt, "[Image 1]、[Image 2]")
	require.Equal(t, []AliVideoMedia{
		{Type: "reference_image", URL: "https://example.com/product.png"},
		{Type: "reference_image", URL: "https://example.com/detail.png"},
	}, aliReq.Input.Media)
	require.Empty(t, aliReq.Input.ImgURL)
}

func TestProcessAliOtherRatiosPricesWan27AliasesAt1080P(t *testing.T) {
	for _, modelName := range []string{"wan2.6-i2v-flash", "wan2.6-t2v"} {
		t.Run(modelName, func(t *testing.T) {
			ratios, err := ProcessAliOtherRatios(&AliVideoRequest{
				Model: modelName,
				Parameters: &AliVideoParameters{
					Resolution: "1080P",
				},
			})

			require.NoError(t, err)
			require.InDelta(t, 1.0/0.6, ratios["resolution-1080P"], 0.000001)
		})
	}
}

func TestProcessAliOtherRatiosPricesMappedWan27ModelsAt1080P(t *testing.T) {
	for _, modelName := range []string{"wan2.7-i2v-2026-04-25", "wan2.7-t2v-2026-06-12"} {
		t.Run(modelName, func(t *testing.T) {
			ratios, err := ProcessAliOtherRatios(&AliVideoRequest{
				Model: modelName,
				Parameters: &AliVideoParameters{
					Resolution: "1080P",
				},
			})

			require.NoError(t, err)
			require.InDelta(t, 1.0/0.6, ratios["resolution-1080P"], 0.000001)
		})
	}
}

func TestProcessAliOtherRatiosPricesHappyHorseAt1080P(t *testing.T) {
	tests := map[string]float64{
		"happyhorse-1.1-r2v": 0.165026 / 0.123769,
		"happyhorse-1.0-r2v": 0.220034 / 0.123769,
	}
	for modelName, want := range tests {
		t.Run(modelName, func(t *testing.T) {
			ratios, err := ProcessAliOtherRatios(&AliVideoRequest{
				Model: modelName,
				Parameters: &AliVideoParameters{
					Resolution: "1080P",
				},
			})

			require.NoError(t, err)
			require.InDelta(t, want, ratios["resolution-1080P"], 0.000001)
		})
	}
}

func TestConvertToAliRequestAnimateExtractsMediaFromMetadataContent(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:  "wan2.2-animate-mix",
		Prompt: "把视频里的人换成图片人物",
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":      "video_url",
					"role":      "reference_video",
					"video_url": map[string]interface{}{"url": "https://example.com/source.mp4"},
				},
				map[string]interface{}{
					"type":      "image_url",
					"role":      "reference_image",
					"image_url": map[string]interface{}{"url": "https://example.com/person.png"},
				},
				map[string]interface{}{"type": "text", "text": "可选描述"},
			},
			"mode":       "wan-pro",
			"resolution": "720p",
			"duration":   5,
			"parameters": map[string]interface{}{
				"resolution": "720P",
				"duration":   5,
				"watermark":  true,
			},
		},
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, "wan2.2-animate-mix", aliReq.Model)
	require.Equal(t, "https://example.com/source.mp4", aliReq.Input.VideoURL)
	require.Equal(t, "https://example.com/person.png", aliReq.Input.ImageURL)
	require.Equal(t, "wan-pro", aliReq.Parameters.Mode)
	// watermark 从 parameters 搬到 input（上游协议位置）
	require.True(t, aliReq.Input.Watermark)
	// duration 保留在内部结构供计费使用
	require.Equal(t, 5, aliReq.Parameters.Duration)

	// 上行请求体只保留 animate 协议字段
	wire := sanitizeAnimateRequest(aliReq)
	body, err := common.Marshal(wire)
	require.NoError(t, err)
	require.Contains(t, string(body), `"video_url":"https://example.com/source.mp4"`)
	require.Contains(t, string(body), `"image_url":"https://example.com/person.png"`)
	require.Contains(t, string(body), `"mode":"wan-pro"`)
	require.Contains(t, string(body), `"watermark":true`)
	require.NotContains(t, string(body), `"duration"`)
	require.NotContains(t, string(body), `"resolution"`)
	require.NotContains(t, string(body), `"prompt_extend"`)
	require.NotContains(t, string(body), `"img_url"`)
	require.NotContains(t, string(body), `"prompt"`)
}

func TestConvertToAliRequestAnimateDefaultsModeToWanStd(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model: "wan2.2-animate-move",
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":      "image_url",
					"image_url": map[string]interface{}{"url": "https://example.com/person.png"},
				},
				map[string]interface{}{
					"type":      "video_url",
					"video_url": map[string]interface{}{"url": "https://example.com/source.mp4"},
				},
			},
		},
	}

	aliReq, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.NoError(t, err)
	require.Equal(t, "wan-std", aliReq.Parameters.Mode)
	require.False(t, aliReq.Input.Watermark)
}

func TestConvertToAliRequestAnimateRequiresVideoAndImage(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model: "wan2.2-animate-mix",
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":      "image_url",
					"image_url": map[string]interface{}{"url": "https://example.com/person.png"},
				},
			},
		},
	}

	_, err := adaptor.convertToAliRequest(testRelayInfo(), req)

	require.Error(t, err)
	require.Contains(t, err.Error(), "video_url and image_url")
}

func TestBuildRequestURLAnimateUsesImage2VideoEndpoint(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://dashscope.aliyuncs.com"}

	info := testRelayInfo()
	info.UpstreamModelName = "wan2.2-animate-move"
	url, err := adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://dashscope.aliyuncs.com/api/v1/services/aigc/image2video/video-synthesis", url)

	info.UpstreamModelName = "wan2.6-i2v"
	url, err = adaptor.BuildRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis", url)
}

func TestParseTaskResultAnimateReadsResultsVideoURL(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"request_id": "req-1",
		"output": {
			"task_id": "task-1",
			"task_status": "SUCCEEDED",
			"results": {"video_url": "https://oss.example.com/out.mp4"}
		},
		"usage": {"video_duration": 5.2, "video_ratio": "standard"}
	}`)

	result, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	require.Equal(t, model.TaskStatusSuccess, result.Status)
	require.Equal(t, "https://oss.example.com/out.mp4", result.Url)
}

func TestParseTaskResultLegacyOutputVideoURLUnchanged(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"request_id": "req-1",
		"output": {
			"task_id": "task-1",
			"task_status": "SUCCEEDED",
			"video_url": "https://oss.example.com/legacy.mp4"
		}
	}`)

	result, err := adaptor.ParseTaskResult(body)

	require.NoError(t, err)
	require.Equal(t, model.TaskStatusSuccess, result.Status)
	require.Equal(t, "https://oss.example.com/legacy.mp4", result.Url)
}

func TestProcessAliOtherRatiosAnimateModePricing(t *testing.T) {
	ratios, err := ProcessAliOtherRatios(&AliVideoRequest{
		Model:      "wan2.2-animate-move",
		Parameters: &AliVideoParameters{Mode: "wan-pro"},
	})
	require.NoError(t, err)
	require.InDelta(t, 1.5, ratios["mode-wan-pro"], 0.000001)

	ratios, err = ProcessAliOtherRatios(&AliVideoRequest{
		Model:      "wan2.2-animate-mix",
		Parameters: &AliVideoParameters{Mode: "wan-std"},
	})
	require.NoError(t, err)
	require.Empty(t, ratios)
}
