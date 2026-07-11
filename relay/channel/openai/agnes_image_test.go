package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertAgnesImageRequestPreservesReferenceAndNestsResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	request := dto.ImageRequest{
		Model:          "agnes-image-2.1-flash",
		Prompt:         "preserve the product",
		Size:           "1K",
		ResponseFormat: "url",
		Extra: map[string]json.RawMessage{
			"ratio":      json.RawMessage(`"9:16"`),
			"extra_body": json.RawMessage(`{"image":["data:image/png;base64,aW1hZ2U="],"response_format":"b64_json"}`),
		},
	}

	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: request.Model}}
	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, request)
	require.NoError(t, err)
	payload := converted.(map[string]json.RawMessage)
	require.NotContains(t, payload, "response_format")
	require.JSONEq(t, `"9:16"`, string(payload["ratio"]))

	var extraBody map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload["extra_body"], &extraBody))
	require.JSONEq(t, `["data:image/png;base64,aW1hZ2U="]`, string(extraBody["image"]))
	require.JSONEq(t, `"b64_json"`, string(extraBody["response_format"]))
}

func TestConvertAgnesImageRequestMovesTopLevelResponseFormat(t *testing.T) {
	request := dto.ImageRequest{
		Model:          "agnes-image-2.0-flash",
		Prompt:         "product photo",
		Size:           "1K",
		ResponseFormat: "url",
	}

	converted, err := convertAgnesImageRequest(request)
	require.NoError(t, err)
	require.NotContains(t, converted, "response_format")

	var extraBody map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(converted["extra_body"], &extraBody))
	require.JSONEq(t, `"url"`, string(extraBody["response_format"]))
}
