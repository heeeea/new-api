package hailuo

const (
	ChannelName = "hailuo-video"
)

var ModelList = []string{
	"MiniMax-H3",
	"MiniMax-Hailuo-2.3",
	"MiniMax-Hailuo-2.3-Fast",
	"MiniMax-Hailuo-02",
	"T2V-01-Director",
	"T2V-01",
	"I2V-01-Director",
	"I2V-01-live",
	"I2V-01",
	"S2V-01",
}

const (
	TextToVideoEndpoint = "/v1/video_generation"
	QueryTaskEndpoint   = "/v1/query/video_generation"
	// V2 接口（MiniMax-H3）
	TextToVideoEndpointV2 = "/v2/video_generation"
	QueryTaskEndpointV2   = "/v2/query/video_generation/"
)

// UpstreamV2TaskIDPrefix 标记走 V2 接口的上游任务 ID，轮询时据此选择 V2 查询端点
const UpstreamV2TaskIDPrefix = "v2:"

const (
	StatusSuccess    = 0
	StatusRateLimit  = 1002
	StatusAuthFailed = 1004
	StatusNoBalance  = 1008
	StatusSensitive  = 1026
	StatusParamError = 2013
	StatusInvalidKey = 2049
)

const (
	TaskStatusPreparing  = "Preparing"
	TaskStatusQueueing   = "Queueing"
	TaskStatusProcessing = "Processing"
	TaskStatusSuccess    = "Success"
	TaskStatusFailed     = "Fail"
)

// V2 接口任务状态（MiniMax-H3）
const (
	TaskStatusV2Queued    = "queued"
	TaskStatusV2Running   = "running"
	TaskStatusV2Succeeded = "succeeded"
	TaskStatusV2Failed    = "failed"
	TaskStatusV2Cancelled = "cancelled"
)

const (
	Resolution512P  = "512P"
	Resolution720P  = "720P"
	Resolution768P  = "768P"
	Resolution1080P = "1080P"
	Resolution2K    = "2K"
)

// MiniMax-H3（V2）时长限制：4-15 秒整数
const (
	H3MinDuration = 4
	H3MaxDuration = 15
)

// H3Resolution2KRatio 2K 档相对 768P 档的计费倍率（官方定价 768P ¥0.5/s、2K ¥0.8/s）
const H3Resolution2KRatio = 1.6

const (
	DefaultDuration   = 6
	DefaultResolution = Resolution720P
)
