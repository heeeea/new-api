package tencentmps

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	appcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveModel(t *testing.T) {
	name, version, ok := resolveModel("kling-omni")
	require.True(t, ok)
	require.Equal(t, "Kling", name)
	require.Equal(t, "3.0-Omni", version)
}

func TestTencentMPSRequestSignature(t *testing.T) {
	body := []byte(`{"TaskId":"task-1"}`)
	req, err := http.NewRequest(http.MethodPost, "https://mps.tencentcloudapi.com", nil)
	require.NoError(t, err)
	require.NoError(t, setSignedHeaders(req, "secret-id|secret-key", "DescribeAigcVideoTask", body))
	require.Contains(t, req.Header.Get("Authorization"), "Credential=secret-id/")
	require.Equal(t, "DescribeAigcVideoTask", req.Header.Get("X-TC-Action"))
}

func TestParseTencentMPSResult(t *testing.T) {
	body := []byte(`{"Response":{"Status":"DONE","VideoUrls":["https://cdn.example/video.mp4"]}}`)
	info, err := (&TaskAdaptor{}).ParseTaskResult(body)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatusSuccess, info.Status)
	require.Equal(t, "https://cdn.example/video.mp4", info.Url)
}

func TestBuildTencentMPSRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a := &TaskAdaptor{}
	task := relaycommon.TaskSubmitReq{Model: "viduq3-turbo", Prompt: "product rotates", Duration: 5, Images: []string{"https://cdn.example/input.png"}, Metadata: map[string]any{"ratio": "9:16"}}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", task)
	info := &relaycommon.RelayInfo{OriginModelName: "viduq3-turbo"}
	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, appcommon.Unmarshal(body, &payload))
	require.Equal(t, "Vidu", payload["ModelName"])
	require.Equal(t, "q3-turbo", payload["ModelVersion"])
	require.Equal(t, "https://cdn.example/input.png", payload["ImageUrl"])
}
