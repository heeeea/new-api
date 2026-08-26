package hailuo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/relay/channel"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
)

// https://platform.minimaxi.com/docs/api-reference/video-generation-intro
type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if taskErr := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); taskErr != nil {
		return taskErr
	}
	// MiniMax-H3（V2）时长限制 4-15 秒整数；此时模型映射尚未执行，按原始模型名判断，
	// 经映射后才是 H3 的请求由 convertToV2RequestPayload 再次校验
	if !isH3Model(info.OriginModelName) && !isH3Model(info.UpstreamModelName) {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapper(err, "invalid_request", http.StatusBadRequest)
	}
	duration := h3RequestDuration(req)
	if duration < H3MinDuration || duration > H3MaxDuration {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("MiniMax-H3 duration must be an integer between %d and %d seconds", H3MinDuration, H3MaxDuration),
			"invalid_duration", http.StatusBadRequest)
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if isH3Model(info.UpstreamModelName) {
		return fmt.Sprintf("%s%s", a.baseURL, TextToVideoEndpointV2), nil
	}
	return fmt.Sprintf("%s%s", a.baseURL, TextToVideoEndpoint), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return nil, fmt.Errorf("request not found in context")
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return nil, fmt.Errorf("invalid request type in context")
	}

	if isH3Model(info.UpstreamModelName) {
		v2Body, err := a.convertToV2RequestPayload(&req, info)
		if err != nil {
			return nil, errors.Wrap(err, "convert request payload failed")
		}
		data, err := common.Marshal(v2Body)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(data), nil
	}

	body, err := a.convertToRequestPayload(&req, info)
	if err != nil {
		return nil, errors.Wrap(err, "convert request payload failed")
	}

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	var hResp VideoResponse
	if err := common.Unmarshal(responseBody, &hResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	// V2 接口（MiniMax-H3）成功响应只有 {"task_id": "..."}，无 base_resp
	isV2 := isH3Model(info.UpstreamModelName)
	if !isV2 && hResp.BaseResp.StatusCode != StatusSuccess {
		taskErr = service.TaskErrorWrapper(
			fmt.Errorf("hailuo api error: %s", hResp.BaseResp.StatusMsg),
			strconv.Itoa(hResp.BaseResp.StatusCode),
			http.StatusBadRequest,
		)
		return
	}

	if hResp.TaskID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName

	c.JSON(http.StatusOK, ov)
	if isV2 {
		// 加前缀标记，轮询时据此走 V2 查询端点
		return UpstreamV2TaskIDPrefix + hResp.TaskID, responseBody, nil
	}
	return hResp.TaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	rawID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	var uri string
	if strings.HasPrefix(rawID, UpstreamV2TaskIDPrefix) {
		// V2（MiniMax-H3）：GET /v2/query/video_generation/{task_id}
		uri = fmt.Sprintf("%s%s%s", baseUrl, QueryTaskEndpointV2, strings.TrimPrefix(rawID, UpstreamV2TaskIDPrefix))
	} else {
		uri = fmt.Sprintf("%s%s?task_id=%s", baseUrl, QueryTaskEndpoint, rawID)
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

// EstimateBilling 返回计费 OtherRatios：seconds（任务时长）+ 分辨率倍率。
// 所有海螺视频模型官方均按秒计价，ModelPrice 配置为每秒单价；
// MiniMax-H3 的 2K 档按官方价（768P ¥0.5/s、2K ¥0.8/s）再加 1.6 倍分辨率倍率。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	if isH3Model(info.UpstreamModelName) {
		v2Req, err := a.convertToV2RequestPayload(&taskReq, info)
		if err != nil {
			return nil
		}
		// duration 已经过 4-15 校验，仍按计费乘数口径钳制一次
		otherRatios := map[string]float64{
			"seconds": float64(min(v2Req.Duration, relaycommon.MaxTaskDurationSeconds)),
		}
		if v2Req.Resolution == Resolution2K {
			otherRatios["resolution"] = H3Resolution2KRatio
		}
		return otherRatios
	}

	// 老的 Hailuo V1 模型：仅加 seconds 乘数
	payload, err := a.convertToRequestPayload(&taskReq, info)
	if err != nil {
		return nil
	}
	// metadata 可覆盖 duration，作为计费乘数前必须钳制
	duration := DefaultDuration
	if payload.Duration != nil && *payload.Duration > 0 {
		duration = min(*payload.Duration, relaycommon.MaxTaskDurationSeconds)
	}
	return map[string]float64{
		"seconds": float64(duration),
	}
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*VideoRequest, error) {
	modelConfig := GetModelConfig(info.UpstreamModelName)
	duration := DefaultDuration
	if req.Duration > 0 {
		duration = req.Duration
	}
	resolution := modelConfig.DefaultResolution
	if req.Size != "" {
		resolution = a.parseResolutionFromSize(req.Size, modelConfig)
	}

	videoRequest := &VideoRequest{
		Model:      info.UpstreamModelName,
		Prompt:     req.Prompt,
		Duration:   &duration,
		Resolution: resolution,
	}
	if err := req.UnmarshalMetadata(&videoRequest); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata to video request failed")
	}

	return videoRequest, nil
}

func (a *TaskAdaptor) parseResolutionFromSize(size string, modelConfig ModelConfig) string {
	switch {
	case strings.Contains(size, "1080"):
		return Resolution1080P
	case strings.Contains(size, "768"):
		return Resolution768P
	case strings.Contains(size, "720"):
		return Resolution720P
	case strings.Contains(size, "512"):
		return Resolution512P
	default:
		return modelConfig.DefaultResolution
	}
}

// ============================
// V2 API（MiniMax-H3）
// ============================

// isH3Model 判断是否只走 V2 接口的 MiniMax-H3 模型
func isH3Model(model string) bool {
	return strings.Contains(strings.ToLower(model), "minimax-h3")
}

// h3RequestDuration 从统一任务请求中取时长（duration 优先，其次 seconds 字符串）
func h3RequestDuration(req relaycommon.TaskSubmitReq) int {
	if req.Duration > 0 {
		return req.Duration
	}
	if seconds, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil {
		return seconds
	}
	return 0
}

// h3Metadata 允许通过 metadata 覆盖/透传的 V2 参数
type h3Metadata struct {
	Duration      *int            `json:"duration"`
	Resolution    *string         `json:"resolution"`
	Ratio         *string         `json:"ratio"`
	AigcWatermark *bool           `json:"aigc_watermark"`
	CallbackURL   *string         `json:"callback_url"`
	Content       []H3ContentItem `json:"content"`
}

var h3AllowedRatios = map[string]bool{
	"adaptive": true,
	"21:9":     true,
	"16:9":     true,
	"4:3":      true,
	"1:1":      true,
	"3:4":      true,
	"9:16":     true,
}

// convertToV2RequestPayload 将统一任务请求映射为 MiniMax-H3 的 V2 请求：
// content 数组 text 必有；首帧图 → first_frame，两张图 → 首尾帧，三张及以上 → reference_image；
// metadata.content 透传 reference_image/reference_video/reference_audio 等厂商扩展项。
func (a *TaskAdaptor) convertToV2RequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*VideoV2Request, error) {
	var meta h3Metadata
	if err := taskcommon.UnmarshalMetadata(req.Metadata, &meta); err != nil {
		return nil, errors.Wrap(err, "unmarshal metadata failed")
	}

	// duration：请求字段优先，metadata 可覆盖；MiniMax-H3 限制 4-15 秒整数
	duration := h3RequestDuration(*req)
	if meta.Duration != nil {
		duration = *meta.Duration
	}
	if duration < H3MinDuration || duration > H3MaxDuration {
		return nil, fmt.Errorf("MiniMax-H3 duration must be an integer between %d and %d seconds", H3MinDuration, H3MaxDuration)
	}

	// resolution：size 含 "2K"/"2k" → 2K，含 "768" → 768P，默认 768P；metadata 可覆盖
	resolution := Resolution768P
	if size := strings.ToLower(req.Size); size != "" {
		if strings.Contains(size, "2k") {
			resolution = Resolution2K
		} else if strings.Contains(size, "768") {
			resolution = Resolution768P
		}
	}
	if meta.Resolution != nil && strings.TrimSpace(*meta.Resolution) != "" {
		resolution = strings.ToUpper(strings.TrimSpace(*meta.Resolution))
	}
	if resolution != Resolution768P && resolution != Resolution2K {
		return nil, fmt.Errorf("MiniMax-H3 resolution must be %s or %s", Resolution768P, Resolution2K)
	}

	// content：text 必有
	content := []H3ContentItem{{Type: "text", Text: req.Prompt}}

	// metadata.content 透传的多模态项（text 项跳过，prompt 已提供）
	metaItems := make([]H3ContentItem, 0, len(meta.Content))
	for _, item := range meta.Content {
		if item.Type == "text" {
			continue
		}
		metaItems = append(metaItems, item)
	}
	hasRefMedia := false
	for _, item := range metaItems {
		if strings.HasPrefix(item.Role, "reference_") {
			hasRefMedia = true
			break
		}
	}

	// 统一请求中的图片输入
	images := make([]string, 0, len(req.Images)+1)
	for _, img := range req.Images {
		if trimmed := strings.TrimSpace(img); trimmed != "" {
			images = append(images, trimmed)
		}
	}
	if len(images) == 0 {
		if trimmed := strings.TrimSpace(req.Image); trimmed != "" {
			images = append(images, trimmed)
		} else if trimmed := strings.TrimSpace(req.InputReference); trimmed != "" {
			images = append(images, trimmed)
		}
	}

	hasFrame := false
	switch {
	case hasRefMedia:
		// 图生（首尾帧）与多模态参考互斥：存在参考素材时，图片一律按参考图
		for _, img := range images {
			content = append(content, H3ContentItem{Type: "image_url", ImageURL: &H3MediaURL{URL: img}, Role: "reference_image"})
		}
	case len(images) == 1:
		content = append(content, H3ContentItem{Type: "image_url", ImageURL: &H3MediaURL{URL: images[0]}, Role: "first_frame"})
		hasFrame = true
	case len(images) == 2:
		content = append(content,
			H3ContentItem{Type: "image_url", ImageURL: &H3MediaURL{URL: images[0]}, Role: "first_frame"},
			H3ContentItem{Type: "image_url", ImageURL: &H3MediaURL{URL: images[1]}, Role: "last_frame"},
		)
		hasFrame = true
	case len(images) > 2:
		// 三张及以上按多模态参考图处理
		for _, img := range images {
			content = append(content, H3ContentItem{Type: "image_url", ImageURL: &H3MediaURL{URL: img}, Role: "reference_image"})
		}
		hasRefMedia = true
	}
	content = append(content, metaItems...)
	for _, item := range metaItems {
		if item.Role == "first_frame" || item.Role == "last_frame" {
			hasFrame = true
		}
	}
	if hasRefMedia && hasFrame {
		return nil, fmt.Errorf("MiniMax-H3 first_frame/last_frame 与 reference_* 输入互斥，不能混用")
	}

	// ratio：文生必须给具体比例（默认 16:9），图生/多模态默认 adaptive
	ratio := ""
	if meta.Ratio != nil {
		ratio = strings.TrimSpace(*meta.Ratio)
	}
	isTextToVideo := len(content) == 1
	if ratio == "" || (isTextToVideo && ratio == "adaptive") {
		if isTextToVideo {
			ratio = "16:9"
		} else {
			ratio = "adaptive"
		}
	}
	if !h3AllowedRatios[ratio] {
		return nil, fmt.Errorf("MiniMax-H3 ratio is invalid: %s", ratio)
	}

	v2Req := &VideoV2Request{
		// 下游模型名可能经映射变为小写，上游只认 "MiniMax-H3"，这里统一回正
		Model:         "MiniMax-H3",
		Content:       content,
		Resolution:    resolution,
		Duration:      duration,
		Ratio:         ratio,
		AigcWatermark: meta.AigcWatermark,
	}
	if meta.CallbackURL != nil {
		v2Req.CallbackURL = *meta.CallbackURL
	}
	return v2Req, nil
}

// parseTaskResultV2 解析 V2 查询接口（GET /v2/query/video_generation/{task_id}）的响应
func parseTaskResultV2(respBody []byte) (*relaycommon.TaskInfo, error) {
	resTask := QueryTaskV2Response{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}
	if resTask.Task == nil {
		return nil, errors.New("v2 task result missing task field")
	}

	task := resTask.Task
	taskResult := relaycommon.TaskInfo{Code: 0, TaskID: task.ID}

	switch task.Status {
	case TaskStatusV2Queued:
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = "20%"
	case TaskStatusV2Running:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "50%"
	case TaskStatusV2Succeeded:
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		taskResult.Url = task.Content.URL
	case TaskStatusV2Failed, TaskStatusV2Cancelled:
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = "task failed"
		if task.Error != nil && task.Error.Message != "" {
			taskResult.Reason = task.Error.Message
		}
	default:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	// V2（MiniMax-H3）查询响应为 {"task": {...}}，按结构探测后走 V2 解析
	var probe struct {
		Task json.RawMessage `json:"task"`
	}
	if err := common.Unmarshal(respBody, &probe); err == nil && len(probe.Task) > 0 && string(probe.Task) != "null" {
		return parseTaskResultV2(respBody)
	}

	resTask := QueryTaskResponse{}
	if err := common.Unmarshal(respBody, &resTask); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{}

	if resTask.BaseResp.StatusCode == StatusSuccess {
		taskResult.Code = 0
	} else {
		taskResult.Code = resTask.BaseResp.StatusCode
		taskResult.Reason = resTask.BaseResp.StatusMsg
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
	}

	switch resTask.Status {
	case TaskStatusPreparing, TaskStatusQueueing, TaskStatusProcessing:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
		if resTask.Status == TaskStatusProcessing {
			taskResult.Progress = "50%"
		}
	case TaskStatusSuccess:
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		taskResult.Url = a.buildVideoURL(resTask.TaskID, resTask.FileID)
	case TaskStatusFailed:
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		if taskResult.Reason == "" {
			taskResult.Reason = "task failed"
		}
	default:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	// V2（MiniMax-H3）任务数据为 {"task": {...}} 结构
	var v2Resp QueryTaskV2Response
	if err := common.Unmarshal(originTask.Data, &v2Resp); err == nil && v2Resp.Task != nil {
		openAIVideo := originTask.ToOpenAIVideo()
		if v2Resp.Task.Error != nil {
			openAIVideo.Error = &dto.OpenAIVideoError{
				Message: v2Resp.Task.Error.Message,
				Code:    v2Resp.Task.Error.Code,
			}
		}
		if v2Resp.Task.Content.URL != "" {
			openAIVideo.SetMetadata("url", v2Resp.Task.Content.URL)
		}
		return common.Marshal(openAIVideo)
	}

	var hailuoResp QueryTaskResponse
	if err := common.Unmarshal(originTask.Data, &hailuoResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal hailuo task data failed")
	}

	openAIVideo := originTask.ToOpenAIVideo()
	if hailuoResp.BaseResp.StatusCode != StatusSuccess {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Message: hailuoResp.BaseResp.StatusMsg,
			Code:    strconv.Itoa(hailuoResp.BaseResp.StatusCode),
		}
	}

	jsonData, err := common.Marshal(openAIVideo)
	if err != nil {
		return nil, errors.Wrap(err, "marshal openai video failed")
	}

	return jsonData, nil
}

func (a *TaskAdaptor) buildVideoURL(_, fileID string) string {
	if a.apiKey == "" || a.baseURL == "" {
		return ""
	}

	url := fmt.Sprintf("%s/v1/files/retrieve?file_id=%s", a.baseURL, fileID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := service.GetHttpClient().Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var retrieveResp RetrieveFileResponse
	if err := common.Unmarshal(responseBody, &retrieveResp); err != nil {
		return ""
	}

	if retrieveResp.BaseResp.StatusCode != StatusSuccess {
		return ""
	}

	return retrieveResp.File.DownloadURL
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsInt(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
