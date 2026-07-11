package tencentmps

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	serviceName   = "mps"
	apiVersion    = "2019-06-12"
	defaultHost   = "mps.tencentcloudapi.com"
	defaultRegion = "ap-guangzhou"
)

type TaskAdaptor struct {
	taskcommon.BaseBilling
	baseURL string
	apiKey  string
}

type createResponse struct {
	Response struct {
		TaskID    string `json:"TaskId"`
		RequestID string `json:"RequestId"`
		Error     *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error,omitempty"`
	} `json:"Response"`
}

type taskResponse struct {
	Response struct {
		Status    string   `json:"Status"`
		Message   string   `json:"Message"`
		VideoURLs []string `json:"VideoUrls"`
		Error     *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error,omitempty"`
	} `json:"Response"`
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
	if a.baseURL == "" {
		a.baseURL = "https://" + defaultHost
	}
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	duration := req.Duration
	if duration <= 0 && req.Seconds != "" {
		duration, _ = strconv.Atoi(req.Seconds)
	}
	if duration <= 0 {
		duration = 5
	}
	return map[string]float64{"seconds": float64(duration)}
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return a.baseURL, nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return setSignedHeaders(req, a.apiKey, "CreateAigcVideoTask", body)
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	modelName, modelVersion, ok := resolveModel(info.OriginModelName)
	if !ok {
		return nil, fmt.Errorf("unsupported Tencent MPS model: %s", info.OriginModelName)
	}
	duration := req.Duration
	if duration <= 0 && req.Seconds != "" {
		duration, _ = strconv.Atoi(req.Seconds)
	}
	if duration <= 0 {
		duration = 5
	}
	resolution := strings.ToUpper(req.Size)
	if resolution == "" || strings.Contains(resolution, "X") {
		resolution = "720P"
	}
	ratio := "9:16"
	if value, ok := req.Metadata["ratio"].(string); ok && value != "" {
		ratio = value
	}
	payload := map[string]any{
		"ModelName":     modelName,
		"ModelVersion":  modelVersion,
		"Prompt":        req.Prompt,
		"Duration":      duration,
		"EnhancePrompt": false,
		"ExtraParameters": map[string]any{
			"Resolution":  resolution,
			"AspectRatio": ratio,
			"LogoAdd":     0,
		},
		"Operator": "yingxiaotai-new-api",
	}
	images := append([]string(nil), req.Images...)
	if len(images) == 0 && req.Image != "" {
		images = append(images, req.Image)
	}
	if len(images) > 0 {
		payload["ImageUrl"] = images[0]
	}
	if len(images) > 1 && supportsLastFrame(modelName) {
		payload["LastImageUrl"] = images[1]
	}
	if enabled, ok := req.Metadata["generate_audio"].(bool); ok && enabled {
		payload["ExtraParameters"].(map[string]any)["EnableAudio"] = true
	}
	body, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(body), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	var result createResponse
	if err := common.Unmarshal(body, &result); err != nil {
		return "", body, service.TaskErrorWrapper(errors.Wrap(err, string(body)), "unmarshal_response_failed", http.StatusBadGateway)
	}
	if result.Response.Error != nil {
		return "", body, service.TaskErrorWrapperLocal(fmt.Errorf("%s: %s", result.Response.Error.Code, result.Response.Error.Message), result.Response.Error.Code, http.StatusBadGateway)
	}
	if result.Response.TaskID == "" {
		return "", body, service.TaskErrorWrapperLocal(errors.New("Tencent MPS did not return TaskId"), "missing_task_id", http.StatusBadGateway)
	}
	video := dto.NewOpenAIVideo()
	video.ID = info.PublicTaskID
	video.TaskID = info.PublicTaskID
	video.Model = info.OriginModelName
	video.CreatedAt = time.Now().Unix()
	c.JSON(http.StatusOK, video)
	return result.Response.TaskID, body, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || taskID == "" {
		return nil, errors.New("invalid task_id")
	}
	payload, err := common.Marshal(map[string]any{"TaskId": taskID})
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(baseURL, "/")
	if endpoint == "" {
		endpoint = "https://" + defaultHost
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if err := setSignedHeaders(req, key, "DescribeAigcVideoTask", payload); err != nil {
		return nil, err
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error) {
	var result taskResponse
	if err := common.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Response.Error != nil {
		return nil, fmt.Errorf("%s: %s", result.Response.Error.Code, result.Response.Error.Message)
	}
	info := &relaycommon.TaskInfo{}
	switch strings.ToUpper(result.Response.Status) {
	case "DONE", "SUCCESS", "SUCCEEDED":
		info.Status = model.TaskStatusSuccess
		info.Progress = "100%"
		if len(result.Response.VideoURLs) > 0 {
			info.Url = result.Response.VideoURLs[0]
		}
	case "FAIL", "FAILED":
		info.Status = model.TaskStatusFailure
		info.Reason = result.Response.Message
	case "PROCESSING", "RUNNING":
		info.Status = model.TaskStatusInProgress
		info.Progress = "30%"
	default:
		info.Status = model.TaskStatusSubmitted
		info.Progress = "10%"
	}
	return info, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var result taskResponse
	if len(task.Data) > 0 {
		_ = common.Unmarshal(task.Data, &result)
	}
	video := dto.NewOpenAIVideo()
	video.ID = task.TaskID
	video.TaskID = task.TaskID
	video.Status = task.Status.ToVideoStatus()
	video.SetProgressStr(task.Progress)
	video.CreatedAt = task.CreatedAt
	video.CompletedAt = task.UpdatedAt
	if len(result.Response.VideoURLs) > 0 {
		video.SetMetadata("url", result.Response.VideoURLs[0])
	}
	if task.Status == model.TaskStatusFailure {
		video.Error = &dto.OpenAIVideoError{Message: result.Response.Message, Code: "tencent_mps_failed"}
	}
	return common.Marshal(video)
}

func (a *TaskAdaptor) GetModelList() []string {
	return []string{"viduq3-turbo", "viduq3-pro", "kling-3.0", "kling-o1", "kling-omni", "hailuo-2.3-fast", "hunyuan-video", "pixverse-v6"}
}

func (a *TaskAdaptor) GetChannelName() string { return "tencent-mps" }

func resolveModel(modelName string) (string, string, bool) {
	switch strings.ToLower(modelName) {
	case "viduq3-turbo":
		return "Vidu", "q3-turbo", true
	case "viduq3-pro":
		return "Vidu", "q3-pro", true
	case "kling-3.0":
		return "Kling", "3.0", true
	case "kling-o1":
		return "Kling", "O1", true
	case "kling-omni":
		return "Kling", "3.0-Omni", true
	case "hailuo-2.3-fast":
		return "Hailuo", "2.3-fast", true
	case "hunyuan-video":
		return "Hunyuan", "1.5", true
	case "pixverse-v6":
		return "PixVerse", "v6", true
	default:
		return "", "", false
	}
}

func supportsLastFrame(modelName string) bool {
	value := strings.ToLower(modelName)
	return strings.Contains(value, "kling") || strings.Contains(value, "vidu") || strings.Contains(value, "pixverse")
}

func setSignedHeaders(req *http.Request, apiKey, action string, body []byte) error {
	_, secretID, secretKey, err := parseKey(apiKey)
	if err != nil {
		return err
	}
	host := req.URL.Host
	timestamp := time.Now().Unix()
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	contentType := "application/json"
	canonicalHeaders := "content-type:" + contentType + "\nhost:" + host + "\nx-tc-action:" + strings.ToLower(action) + "\n"
	signedHeaders := "content-type;host;x-tc-action"
	canonicalRequest := strings.Join([]string{"POST", "/", "", canonicalHeaders, signedHeaders, sha256Hex(body)}, "\n")
	algorithm := "TC3-HMAC-SHA256"
	scope := date + "/" + serviceName + "/tc3_request"
	stringToSign := strings.Join([]string{algorithm, strconv.FormatInt(timestamp, 10), scope, sha256Hex([]byte(canonicalRequest))}, "\n")
	secretDate := hmacSHA256([]byte("TC3"+secretKey), date)
	secretService := hmacSHA256(secretDate, serviceName)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s", algorithm, secretID, scope, signedHeaders, signature))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-TC-Version", apiVersion)
	req.Header.Set("X-TC-Region", defaultRegion)
	return nil
}

func parseKey(value string) (string, string, string, error) {
	parts := strings.Split(strings.TrimPrefix(value, "Bearer "), "|")
	if len(parts) == 2 {
		return "", parts[0], parts[1], nil
	}
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2], nil
	}
	return "", "", "", errors.New("Tencent key must be SecretId|SecretKey or AppId|SecretId|SecretKey")
}

func sha256Hex(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key []byte, value string) []byte {
	hash := hmac.New(sha256.New, key)
	_, _ = hash.Write([]byte(value))
	return hash.Sum(nil)
}
