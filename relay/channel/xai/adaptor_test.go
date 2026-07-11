package xai

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

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
