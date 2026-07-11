package tencent

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertTencentImageRequestSignsTextToImageLite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	c.Set(string(constant.ContextKeyChannelKey), "0|secret-id|secret-key")
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesGenerations}
	adaptor := &Adaptor{}
	adaptor.Init(info)

	converted, err := adaptor.ConvertImageRequest(c, info, dto.ImageRequest{Model: "hunyuan-image", Prompt: "product photo"})

	require.NoError(t, err)
	require.Equal(t, TencentImageRequest{Prompt: "product photo"}, converted)
	require.Equal(t, "TextToImageLite", adaptor.Action)
	require.Contains(t, adaptor.Sign, "Credential=secret-id/")
}

func TestTencentImageHandlerNormalizesURL(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(`{"Response":{"RequestId":"req-1","ResultImage":"https://cdn.example/image.png"}}`)),
	}

	usage, apiErr := tencentImageHandler(c, resp)

	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	var result dto.ImageResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	require.Len(t, result.Data, 1)
	require.Equal(t, "https://cdn.example/image.png", result.Data[0].Url)
}
