package xai

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertImageRequestMapsPortraitSizeToAspectRatio(t *testing.T) {
	t.Parallel()

	converted, err := (&Adaptor{}).ConvertImageRequest(
		gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New()),
		&relaycommon.RelayInfo{},
		dto.ImageRequest{
			Model:  "grok-imagine-image",
			Prompt: "portrait product photo",
			Size:   "576x1024",
		},
	)
	require.NoError(t, err)

	request, ok := converted.(ImageRequest)
	require.True(t, ok)
	assert.Equal(t, "9:16", request.AspectRatio)
}

func TestConvertImageRequestAddsOfficialXAIResolutionAndInputPriceRatio(t *testing.T) {
	t.Parallel()

	info := &relaycommon.RelayInfo{PriceData: types.PriceData{UsePrice: true, ModelPrice: 0.05}}
	_, err := (&Adaptor{}).ConvertImageRequest(
		gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New()),
		info,
		dto.ImageRequest{
			Model:      "grok-imagine-image-quality",
			Prompt:     "put the product on the model",
			Resolution: "2k",
			Images:     json.RawMessage(`[{"type":"image_url","url":"https://cdn.example/product.png"}]`),
		},
	)
	require.NoError(t, err)

	// Official xAI cost: $0.07 2K output + $0.01 per input image,
	// relative to the configured $0.05 1K output base price.
	assert.InDelta(t, 1.6, info.PriceData.OtherRatios()["xai_image_price"], 0.000001)
}

func TestConvertImageEditPreservesExtendedRatioResolutionAndReferences(t *testing.T) {
	t.Parallel()

	converted, err := (&Adaptor{}).ConvertImageRequest(
		gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New()),
		&relaycommon.RelayInfo{},
		dto.ImageRequest{
			Model:       "grok-imagine-image-quality",
			Prompt:      "put the product on the model",
			Quality:     "high",
			AspectRatio: "20:9",
			Images:      json.RawMessage(`[{"type":"image_url","url":"data:image/png;base64,AAAA"},{"type":"image_url","url":"https://cdn.example/product.png"}]`),
		},
	)
	require.NoError(t, err)

	request, ok := converted.(ImageRequest)
	require.True(t, ok)
	assert.Equal(t, "20:9", request.AspectRatio)
	assert.Equal(t, "2k", request.Resolution)
	assert.Nil(t, request.Image)
	assert.Equal(t, []ImageReference{
		{Type: "image_url", URL: "data:image/png;base64,AAAA"},
		{Type: "image_url", URL: "https://cdn.example/product.png"},
	}, request.Images)
}
