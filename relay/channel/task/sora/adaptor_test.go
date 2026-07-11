package sora

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestXAIRequestUsesGenerationEndpointAndProviderFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := &TaskAdaptor{}
	a.Init(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeXai, ChannelBaseUrl: "https://api.x.ai"},
	})
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeXai, ChannelBaseUrl: "https://api.x.ai", UpstreamModelName: "grok-imagine-video"},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}

	url, err := a.BuildRequestURL(info)
	require.NoError(t, err)
	require.Equal(t, "https://api.x.ai/v1/videos/generations", url)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model:    "grok-imagine-video",
		Prompt:   "product rotates",
		Duration: 8,
		Images:   []string{"https://cdn.example/first.png", "https://cdn.example/product.png"},
		Metadata: map[string]any{"ratio": "9:16", "resolution": "720p", "mode": "image_reference"},
	})

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, appcommon.Unmarshal(body, &payload))
	require.Equal(t, "grok-imagine-video", payload["model"])
	require.Equal(t, "product rotates", payload["prompt"])
	require.Equal(t, float64(8), payload["duration"])
	require.Equal(t, "9:16", payload["aspect_ratio"])
	require.Equal(t, "720p", payload["resolution"])
	require.NotContains(t, payload, "image")
	require.Equal(t, []any{
		map[string]any{"url": "https://cdn.example/first.png"},
		map[string]any{"url": "https://cdn.example/product.png"},
	}, payload["reference_images"])
}

func TestXAIResponseAndPollingUseRequestIDAndNestedVideoURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := &TaskAdaptor{}
	a.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeXai}})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"request_id":"xai-request-1"}`))}
	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task-public"}}

	taskID, _, taskErr := a.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	require.Equal(t, "xai-request-1", taskID)
	require.Contains(t, recorder.Body.String(), `"id":"task-public"`)

	result, err := a.ParseTaskResult([]byte(`{"status":"done","video":{"url":"https://vidgen.x.ai/result.mp4","duration":8},"model":"grok-imagine-video"}`))
	require.NoError(t, err)
	require.Equal(t, model.TaskStatusSuccess, result.Status)
	require.Equal(t, "https://vidgen.x.ai/result.mp4", result.Url)

	failed, err := a.ParseTaskResult([]byte(`{"status":"expired"}`))
	require.NoError(t, err)
	require.Equal(t, model.TaskStatusFailure, failed.Status)

	normalized, err := a.ConvertToOpenAIVideo(&model.Task{
		TaskID:      "task-public",
		Platform:    constant.TaskPlatform("48"),
		Status:      model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{ResultURL: "https://vidgen.x.ai/result.mp4"},
		Data:        []byte(`{"status":"done","video":{"url":"https://vidgen.x.ai/result.mp4"}}`),
	})
	require.NoError(t, err)
	var video map[string]any
	require.NoError(t, appcommon.Unmarshal(normalized, &video))
	require.Equal(t, "completed", video["status"])
	require.Equal(t, map[string]any{"url": "https://vidgen.x.ai/result.mp4"}, video["metadata"])
}
