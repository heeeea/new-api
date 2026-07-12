package volcengine

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertSeedream5ImageRequestAddsOfficialLargeOutputAndReferenceInputPrice(t *testing.T) {
	info := &relaycommon.RelayInfo{
		OriginModelName: "seedream-5-0-pro",
		RelayMode:       constant.RelayModeImagesGenerations,
		PriceData:       types.PriceData{UsePrice: true, ModelPrice: 0.041095890411},
	}
	_, err := (&Adaptor{}).ConvertImageRequest(
		gin.CreateTestContextOnly(httptest.NewRecorder(), gin.New()),
		info,
		dto.ImageRequest{
			Model:  "seedream-5-0-pro",
			Prompt: "product photo",
			Size:   "2048x2048",
			Images: json.RawMessage(`[{"url":"https://cdn.example/one.png"},{"url":"https://cdn.example/two.png"}]`),
		},
	)
	require.NoError(t, err)

	// Official Ark cost: ¥0.60 large output + 2 * ¥0.02 reference inputs,
	// relative to the configured ¥0.30 normal-output base price.
	require.InDelta(t, 0.64/0.30, info.PriceData.OtherRatios()["seedream5_image_price"], 0.000001)
}
