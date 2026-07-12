package tencent

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type Adaptor struct {
	Sign      string
	AppID     int64
	Action    string
	Version   string
	Timestamp int64
}

func (a *Adaptor) ConvertGeminiRequest(*gin.Context, *relaycommon.RelayInfo, *dto.GeminiChatRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertClaudeRequest(*gin.Context, *relaycommon.RelayInfo, *dto.ClaudeRequest) (any, error) {
	//TODO implement me
	panic("implement me")
	return nil, nil
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	apiKey := common.GetContextKeyString(c, constant.ContextKeyChannelKey)
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	appID, secretID, secretKey, err := parseTencentConfig(apiKey)
	if err != nil {
		return nil, err
	}
	a.AppID = appID
	resolution := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(request.Size)), "x", ":")
	allowedResolutions := map[string]bool{
		"768:768": true, "768:1024": true, "1024:768": true, "1024:1024": true,
		"720:1280": true, "1280:720": true, "768:1280": true, "1280:768": true,
		"1080:1920": true, "1920:1080": true,
	}
	if !allowedResolutions[resolution] {
		resolution = ""
	}
	tencentRequest := TencentImageRequest{Prompt: request.Prompt, Resolution: resolution, LogoAdd: 0}
	a.Sign, err = getTencentSignForPayload(tencentRequest, a, secretID, secretKey)
	if err != nil {
		return nil, err
	}
	return tencentRequest, nil
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
	a.Action = "ChatCompletions"
	a.Version = "2023-09-01"
	if info.RelayMode == relayconstant.RelayModeImagesGenerations {
		a.Action = "TextToImageLite"
	}
	a.Timestamp = common.GetTimestamp()
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/", info.ChannelBaseUrl), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Header, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	req.Set("Authorization", a.Sign)
	req.Set("X-TC-Action", a.Action)
	req.Set("X-TC-Version", a.Version)
	req.Set("X-TC-Timestamp", strconv.FormatInt(a.Timestamp, 10))
	return nil
}

func (a *Adaptor) ConvertOpenAIRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	apiKey := common.GetContextKeyString(c, constant.ContextKeyChannelKey)
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	appId, secretId, secretKey, err := parseTencentConfig(apiKey)
	a.AppID = appId
	if err != nil {
		return nil, err
	}
	tencentRequest := requestOpenAI2Tencent(a, *request)
	// we have to calculate the sign here
	a.Sign = getTencentSign(*tencentRequest, a, secretId, secretKey)
	return tencentRequest, nil
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) ConvertEmbeddingRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.EmbeddingRequest) (any, error) {
	//TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertOpenAIResponsesRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.OpenAIResponsesRequest) (any, error) {
	// TODO implement me
	return nil, errors.New("not implemented")
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (any, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage any, err *types.NewAPIError) {
	if info.RelayMode == relayconstant.RelayModeImagesGenerations {
		return tencentImageHandler(c, resp)
	}
	if info.IsStream {
		usage, err = tencentStreamHandler(c, info, resp)
	} else {
		usage, err = tencentHandler(c, info, resp)
	}
	return
}

func (a *Adaptor) GetModelList() []string {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
