package relay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	channelconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAudioHelperSanitizesAliHTTPErrorCredential(t *testing.T) {
	service.InitHttpClient()
	const apiKey = "sk-real-http-error-secret"
	for _, statusCode := range []int{http.StatusBadRequest, http.StatusUnauthorized} {
		t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(statusCode)
				_, _ = fmt.Fprintf(w, `{"code":"InvalidApiKey-%s","message":"Authorization Bearer %s is invalid"}`, apiKey, apiKey)
			}))
			defer upstream.Close()

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/speech", nil)
			common.SetContextKey(c, channelconstant.ContextKeyChannelType, channelconstant.ChannelTypeAli)
			common.SetContextKey(c, channelconstant.ContextKeyChannelBaseUrl, upstream.URL)
			common.SetContextKey(c, channelconstant.ContextKeyChannelKey, apiKey)
			common.SetContextKey(c, channelconstant.ContextKeyOriginalModel, "qwen3-tts-flash")

			request := &dto.AudioRequest{Model: "qwen3-tts-flash", Input: "test", Voice: "Cherry"}
			apiErr := AudioHelper(c, &relaycommon.RelayInfo{
				RelayMode:       relayconstant.RelayModeAudioSpeech,
				OriginModelName: request.Model,
				Request:         request,
			})
			require.NotNil(t, apiErr)
			assert.Equal(t, statusCode, apiErr.StatusCode)
			assert.NotContains(t, apiErr.Error(), apiKey)
			assert.NotContains(t, apiErr.ToOpenAIError().Message, apiKey)
		})
	}
}
