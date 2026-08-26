package hailuo

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRelayInfo(upstreamModel string) *relaycommon.RelayInfo {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{},
	}
	info.UpstreamModelName = upstreamModel
	return info
}

func testGinContext(req relaycommon.TaskSubmitReq) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("task_request", req)
	return c
}

func TestIsH3Model(t *testing.T) {
	assert.True(t, isH3Model("MiniMax-H3"))
	assert.True(t, isH3Model("minimax-h3"))
	assert.False(t, isH3Model("MiniMax-Hailuo-2.3"))
	assert.False(t, isH3Model("MiniMax-Hailuo-02"))
	assert.False(t, isH3Model("T2V-01"))
	assert.False(t, isH3Model(""))
}

func TestBuildRequestURLH3UsesV2Endpoint(t *testing.T) {
	adaptor := &TaskAdaptor{baseURL: "https://api.minimaxi.com"}

	url, err := adaptor.BuildRequestURL(testRelayInfo("MiniMax-H3"))
	require.NoError(t, err)
	assert.Equal(t, "https://api.minimaxi.com/v2/video_generation", url)

	url, err = adaptor.BuildRequestURL(testRelayInfo("MiniMax-Hailuo-2.3"))
	require.NoError(t, err)
	assert.Equal(t, "https://api.minimaxi.com/v1/video_generation", url)
}

func TestConvertToV2RequestTextToVideo(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-H3",
		Prompt:   "一个男孩在海边打篮球",
		Size:     "2K",
		Duration: 5,
	}

	v2Req, err := adaptor.convertToV2RequestPayload(&req, testRelayInfo("MiniMax-H3"))
	require.NoError(t, err)
	assert.Equal(t, "MiniMax-H3", v2Req.Model)
	assert.Equal(t, Resolution2K, v2Req.Resolution)
	assert.Equal(t, 5, v2Req.Duration)
	// 文生必须给具体比例
	assert.Equal(t, "16:9", v2Req.Ratio)
	require.Len(t, v2Req.Content, 1)
	assert.Equal(t, "text", v2Req.Content[0].Type)
	assert.Equal(t, "一个男孩在海边打篮球", v2Req.Content[0].Text)
}

func TestConvertToV2RequestImageToVideoFirstLastFrame(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-H3",
		Prompt:   "让画面动起来",
		Images:   []string{"https://example.com/first.png", "https://example.com/last.png"},
		Duration: 6,
	}

	v2Req, err := adaptor.convertToV2RequestPayload(&req, testRelayInfo("MiniMax-H3"))
	require.NoError(t, err)
	assert.Equal(t, Resolution768P, v2Req.Resolution)
	// 图生 ratio 传 adaptive
	assert.Equal(t, "adaptive", v2Req.Ratio)
	require.Len(t, v2Req.Content, 3)
	assert.Equal(t, "first_frame", v2Req.Content[1].Role)
	assert.Equal(t, "https://example.com/first.png", v2Req.Content[1].ImageURL.URL)
	assert.Equal(t, "last_frame", v2Req.Content[2].Role)
	assert.Equal(t, "https://example.com/last.png", v2Req.Content[2].ImageURL.URL)
}

func TestConvertToV2RequestSingleImageIsFirstFrame(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-H3",
		Prompt:   "让画面动起来",
		Image:    "https://example.com/first.png",
		Duration: 6,
	}

	v2Req, err := adaptor.convertToV2RequestPayload(&req, testRelayInfo("MiniMax-H3"))
	require.NoError(t, err)
	require.Len(t, v2Req.Content, 2)
	assert.Equal(t, "first_frame", v2Req.Content[1].Role)
}

func TestConvertToV2RequestReferenceMediaFromMetadata(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-H3",
		Prompt:   "角色说话，音色参考音频",
		Images:   []string{"https://example.com/person.png"},
		Duration: 8,
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":      "video_url",
					"role":      "reference_video",
					"video_url": map[string]interface{}{"url": "https://example.com/ref.mp4"},
				},
				map[string]interface{}{
					"type":      "audio_url",
					"role":      "reference_audio",
					"audio_url": map[string]interface{}{"url": "https://example.com/ref.mp3"},
				},
			},
			"ratio":          "9:16",
			"aigc_watermark": true,
		},
	}

	v2Req, err := adaptor.convertToV2RequestPayload(&req, testRelayInfo("MiniMax-H3"))
	require.NoError(t, err)
	assert.Equal(t, "9:16", v2Req.Ratio)
	require.NotNil(t, v2Req.AigcWatermark)
	assert.True(t, *v2Req.AigcWatermark)
	require.Len(t, v2Req.Content, 4)
	// 存在参考素材时，统一请求的图片按 reference_image 处理
	assert.Equal(t, "reference_image", v2Req.Content[1].Role)
	assert.Equal(t, "reference_video", v2Req.Content[2].Role)
	assert.Equal(t, "https://example.com/ref.mp4", v2Req.Content[2].VideoURL.URL)
	assert.Equal(t, "reference_audio", v2Req.Content[3].Role)
	assert.Equal(t, "https://example.com/ref.mp3", v2Req.Content[3].AudioURL.URL)
}

func TestConvertToV2RequestRejectsMixedFrameAndReference(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-H3",
		Prompt:   "互斥校验",
		Images:   []string{"https://example.com/first.png"},
		Duration: 5,
		Metadata: map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type":      "image_url",
					"role":      "last_frame",
					"image_url": map[string]interface{}{"url": "https://example.com/last.png"},
				},
				map[string]interface{}{
					"type":      "video_url",
					"role":      "reference_video",
					"video_url": map[string]interface{}{"url": "https://example.com/ref.mp4"},
				},
			},
		},
	}

	_, err := adaptor.convertToV2RequestPayload(&req, testRelayInfo("MiniMax-H3"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "互斥")
}

func TestConvertToV2RequestDurationBounds(t *testing.T) {
	adaptor := &TaskAdaptor{}
	for _, duration := range []int{0, 3, 16} {
		req := relaycommon.TaskSubmitReq{
			Model:    "MiniMax-H3",
			Prompt:   "时长校验",
			Duration: duration,
		}
		_, err := adaptor.convertToV2RequestPayload(&req, testRelayInfo("MiniMax-H3"))
		require.Error(t, err, "duration=%d should be rejected", duration)
		assert.Contains(t, err.Error(), "duration")
	}

	// metadata 覆盖 duration 同样受限
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-H3",
		Prompt:   "时长校验",
		Duration: 5,
		Metadata: map[string]interface{}{"duration": 20},
	}
	_, err := adaptor.convertToV2RequestPayload(&req, testRelayInfo("MiniMax-H3"))
	require.Error(t, err)
}

func TestEstimateBillingH3SecondsAndResolution(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-H3",
		Prompt:   "计费估算",
		Size:     "2K",
		Duration: 10,
	}

	ratios := adaptor.EstimateBilling(testGinContext(req), testRelayInfo("MiniMax-H3"))
	require.NotNil(t, ratios)
	assert.InDelta(t, 10.0, ratios["seconds"], 0.000001)
	assert.InDelta(t, 1.6, ratios["resolution"], 0.000001)
}

func TestEstimateBillingH3Default768PNoResolutionRatio(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-H3",
		Prompt:   "计费估算",
		Duration: 4,
	}

	ratios := adaptor.EstimateBilling(testGinContext(req), testRelayInfo("MiniMax-H3"))
	require.NotNil(t, ratios)
	assert.InDelta(t, 4.0, ratios["seconds"], 0.000001)
	_, hasResolution := ratios["resolution"]
	assert.False(t, hasResolution)
}

func TestEstimateBillingH3InvalidDurationReturnsNil(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-H3",
		Prompt:   "计费估算",
		Duration: 30,
	}

	assert.Nil(t, adaptor.EstimateBilling(testGinContext(req), testRelayInfo("MiniMax-H3")))
}

func TestEstimateBillingV1SecondsOnly(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "MiniMax-Hailuo-2.3",
		Prompt:   "计费估算",
		Duration: 10,
	}

	ratios := adaptor.EstimateBilling(testGinContext(req), testRelayInfo("MiniMax-Hailuo-2.3"))
	require.NotNil(t, ratios)
	assert.InDelta(t, 10.0, ratios["seconds"], 0.000001)
	_, hasResolution := ratios["resolution"]
	assert.False(t, hasResolution)
}

func TestParseTaskResultV2Succeeded(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"task": {
			"id": "424010985738629",
			"model": "MiniMax-H3",
			"status": "succeeded",
			"content": {"url": "https://cdn.example.com/output.mp4"},
			"resolution": "2K",
			"duration": 5,
			"usage": {"total_seconds": 5, "output_seconds": 5},
			"ratio": "16:9"
		}
	}`)

	result, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, result.Status)
	assert.Equal(t, "https://cdn.example.com/output.mp4", result.Url)
	assert.Equal(t, "100%", result.Progress)
}

func TestParseTaskResultV2Failed(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"task": {
			"id": "424010985738630",
			"model": "MiniMax-H3",
			"status": "failed",
			"error": {"code": "1026", "message": "video description contains sensitive content"}
		}
	}`)

	result, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusFailure, result.Status)
	assert.Equal(t, "video description contains sensitive content", result.Reason)
}

func TestParseTaskResultV2Running(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{"task": {"id": "424010985738631", "status": "running"}}`)

	result, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, result.Status)
}

// V1 查询响应结构不受 V2 探测影响
func TestParseTaskResultV1Unchanged(t *testing.T) {
	adaptor := &TaskAdaptor{}
	body := []byte(`{
		"task_id": "123456",
		"status": "Processing",
		"base_resp": {"status_code": 0, "status_msg": ""}
	}`)

	result, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, result.Status)
}
