package controller

import (
	"errors"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const envManagedChannelTag = "env-managed"

type envChannelCredentialMode struct {
	AwsKeyType    dto.AwsKeyType
	VertexKeyType dto.VertexKeyType
}

type envChannelDefinition struct {
	Name           string
	ChannelType    int
	KeyEnv         string
	BaseURLEnv     string
	ModelsEnv      string
	OtherEnv       string
	RequiredEnv    []string
	AutoSyncModels bool
	CredentialMode envChannelCredentialMode
}

var officialEnvChannelDefinitions = []envChannelDefinition{
	{Name: "env-openai", ChannelType: constant.ChannelTypeOpenAI, KeyEnv: "OPENAI_API_KEY", BaseURLEnv: "OPENAI_BASE_URL", ModelsEnv: "OPENAI_MODELS", AutoSyncModels: true},
	{Name: "env-openai-video", ChannelType: constant.ChannelTypeSora, KeyEnv: "OPENAI_API_KEY", BaseURLEnv: "OPENAI_BASE_URL", ModelsEnv: "OPENAI_VIDEO_MODELS"},
	{Name: "env-anthropic", ChannelType: constant.ChannelTypeAnthropic, KeyEnv: "ANTHROPIC_API_KEY", BaseURLEnv: "ANTHROPIC_BASE_URL", ModelsEnv: "ANTHROPIC_MODELS"},
	{Name: "env-azure-openai", ChannelType: constant.ChannelTypeAzure, KeyEnv: "AZURE_OPENAI_API_KEY", BaseURLEnv: "AZURE_OPENAI_BASE_URL", ModelsEnv: "AZURE_OPENAI_MODELS", OtherEnv: "AZURE_OPENAI_API_VERSION", RequiredEnv: []string{"AZURE_OPENAI_BASE_URL", "AZURE_OPENAI_MODELS"}},
	{Name: "env-google-gemini", ChannelType: constant.ChannelTypeGemini, KeyEnv: "GEMINI_API_KEY", BaseURLEnv: "GEMINI_BASE_URL", ModelsEnv: "GEMINI_MODELS", OtherEnv: "GEMINI_API_VERSION", AutoSyncModels: true},
	{Name: "env-google-vertex", ChannelType: constant.ChannelTypeVertexAi, KeyEnv: "VERTEX_AI_CREDENTIALS", BaseURLEnv: "VERTEX_AI_BASE_URL", ModelsEnv: "VERTEX_AI_MODELS", OtherEnv: "VERTEX_AI_REGIONS", RequiredEnv: []string{"VERTEX_AI_REGIONS"}, CredentialMode: envChannelCredentialMode{VertexKeyType: dto.VertexKeyTypeJSON}},
	{Name: "env-aws-bedrock", ChannelType: constant.ChannelTypeAws, KeyEnv: "AWS_BEDROCK_CREDENTIALS", ModelsEnv: "AWS_BEDROCK_MODELS", CredentialMode: envChannelCredentialMode{AwsKeyType: dto.AwsKeyTypeAKSK}},
	{Name: "env-deepseek", ChannelType: constant.ChannelTypeDeepSeek, KeyEnv: "DEEPSEEK_API_KEY", BaseURLEnv: "DEEPSEEK_BASE_URL", ModelsEnv: "DEEPSEEK_MODELS", AutoSyncModels: true},
	{Name: "env-xai", ChannelType: constant.ChannelTypeXai, KeyEnv: "XAI_API_KEY", BaseURLEnv: "XAI_BASE_URL", ModelsEnv: "XAI_MODELS", AutoSyncModels: true},
	{Name: "env-alibaba-model-studio", ChannelType: constant.ChannelTypeAli, KeyEnv: "DASHSCOPE_API_KEY", BaseURLEnv: "DASHSCOPE_BASE_URL", ModelsEnv: "DASHSCOPE_MODELS", AutoSyncModels: true},
	{Name: "env-volcengine-ark", ChannelType: constant.ChannelTypeVolcEngine, KeyEnv: "VOLCENGINE_API_KEY", BaseURLEnv: "VOLCENGINE_BASE_URL", ModelsEnv: "VOLCENGINE_MODELS", AutoSyncModels: true},
	{Name: "env-doubao-video", ChannelType: constant.ChannelTypeDoubaoVideo, KeyEnv: "VOLCENGINE_API_KEY", BaseURLEnv: "VOLCENGINE_BASE_URL", ModelsEnv: "DOUBAO_VIDEO_MODELS"},
	{Name: "env-tencent-hunyuan", ChannelType: constant.ChannelTypeTencent, KeyEnv: "TENCENT_HUNYUAN_CREDENTIALS", BaseURLEnv: "TENCENT_HUNYUAN_BASE_URL", ModelsEnv: "TENCENT_HUNYUAN_MODELS"},
	{Name: "env-minimax", ChannelType: constant.ChannelTypeMiniMax, KeyEnv: "MINIMAX_API_KEY", BaseURLEnv: "MINIMAX_BASE_URL", ModelsEnv: "MINIMAX_MODELS", AutoSyncModels: true},
	{Name: "env-moonshot", ChannelType: constant.ChannelTypeMoonshot, KeyEnv: "MOONSHOT_API_KEY", BaseURLEnv: "MOONSHOT_BASE_URL", ModelsEnv: "MOONSHOT_MODELS", AutoSyncModels: true},
	{Name: "env-zhipu", ChannelType: constant.ChannelTypeZhipu_v4, KeyEnv: "ZHIPU_API_KEY", BaseURLEnv: "ZHIPU_BASE_URL", ModelsEnv: "ZHIPU_MODELS", AutoSyncModels: true},
	{Name: "env-baidu-qianfan", ChannelType: constant.ChannelTypeBaiduV2, KeyEnv: "BAIDU_QIANFAN_API_KEY", BaseURLEnv: "BAIDU_QIANFAN_BASE_URL", ModelsEnv: "BAIDU_QIANFAN_MODELS", AutoSyncModels: true},
	{Name: "env-iflytek", ChannelType: constant.ChannelTypeXunfei, KeyEnv: "IFLYTEK_CREDENTIALS", BaseURLEnv: "IFLYTEK_BASE_URL", ModelsEnv: "IFLYTEK_MODELS", OtherEnv: "IFLYTEK_API_VERSION"},
	{Name: "env-360", ChannelType: constant.ChannelType360, KeyEnv: "QIHOO360_API_KEY", BaseURLEnv: "QIHOO360_BASE_URL", ModelsEnv: "QIHOO360_MODELS", AutoSyncModels: true},
	{Name: "env-openrouter", ChannelType: constant.ChannelTypeOpenRouter, KeyEnv: "OPENROUTER_API_KEY", BaseURLEnv: "OPENROUTER_BASE_URL", ModelsEnv: "OPENROUTER_MODELS", AutoSyncModels: true},
	{Name: "env-perplexity", ChannelType: constant.ChannelTypePerplexity, KeyEnv: "PERPLEXITY_API_KEY", BaseURLEnv: "PERPLEXITY_BASE_URL", ModelsEnv: "PERPLEXITY_MODELS", AutoSyncModels: true},
	{Name: "env-lingyi", ChannelType: constant.ChannelTypeLingYiWanWu, KeyEnv: "LINGYI_API_KEY", BaseURLEnv: "LINGYI_BASE_URL", ModelsEnv: "LINGYI_MODELS", AutoSyncModels: true},
	{Name: "env-cohere", ChannelType: constant.ChannelTypeCohere, KeyEnv: "COHERE_API_KEY", BaseURLEnv: "COHERE_BASE_URL", ModelsEnv: "COHERE_MODELS", AutoSyncModels: true},
	{Name: "env-jina", ChannelType: constant.ChannelTypeJina, KeyEnv: "JINA_API_KEY", BaseURLEnv: "JINA_BASE_URL", ModelsEnv: "JINA_MODELS", AutoSyncModels: true},
	{Name: "env-cloudflare-workers-ai", ChannelType: constant.ChannelCloudflare, KeyEnv: "CLOUDFLARE_API_TOKEN", BaseURLEnv: "CLOUDFLARE_BASE_URL", ModelsEnv: "CLOUDFLARE_MODELS", OtherEnv: "CLOUDFLARE_ACCOUNT_ID", RequiredEnv: []string{"CLOUDFLARE_ACCOUNT_ID"}},
	{Name: "env-siliconflow", ChannelType: constant.ChannelTypeSiliconFlow, KeyEnv: "SILICONFLOW_API_KEY", BaseURLEnv: "SILICONFLOW_BASE_URL", ModelsEnv: "SILICONFLOW_MODELS", AutoSyncModels: true},
	{Name: "env-mistral", ChannelType: constant.ChannelTypeMistral, KeyEnv: "MISTRAL_API_KEY", BaseURLEnv: "MISTRAL_BASE_URL", ModelsEnv: "MISTRAL_MODELS", AutoSyncModels: true},
	{Name: "env-coze", ChannelType: constant.ChannelTypeCoze, KeyEnv: "COZE_API_KEY", BaseURLEnv: "COZE_BASE_URL", ModelsEnv: "COZE_MODELS", OtherEnv: "COZE_BOT_ID", RequiredEnv: []string{"COZE_BOT_ID"}},
	{Name: "env-kling", ChannelType: constant.ChannelTypeKling, KeyEnv: "KLING_API_KEY", BaseURLEnv: "KLING_BASE_URL", ModelsEnv: "KLING_MODELS"},
	{Name: "env-jimeng", ChannelType: constant.ChannelTypeJimeng, KeyEnv: "JIMENG_CREDENTIALS", BaseURLEnv: "JIMENG_BASE_URL", ModelsEnv: "JIMENG_MODELS"},
	{Name: "env-vidu", ChannelType: constant.ChannelTypeVidu, KeyEnv: "VIDU_API_KEY", BaseURLEnv: "VIDU_BASE_URL", ModelsEnv: "VIDU_MODELS"},
	{Name: "env-replicate", ChannelType: constant.ChannelTypeReplicate, KeyEnv: "REPLICATE_API_TOKEN", BaseURLEnv: "REPLICATE_BASE_URL", ModelsEnv: "REPLICATE_MODELS"},
	{Name: "env-openai-compatible-1", ChannelType: constant.ChannelTypeOpenAI, KeyEnv: "OPENAI_COMPATIBLE_1_API_KEY", BaseURLEnv: "OPENAI_COMPATIBLE_1_BASE_URL", ModelsEnv: "OPENAI_COMPATIBLE_1_MODELS", RequiredEnv: []string{"OPENAI_COMPATIBLE_1_BASE_URL", "OPENAI_COMPATIBLE_1_MODELS"}, AutoSyncModels: true},
	{Name: "env-openai-compatible-2", ChannelType: constant.ChannelTypeOpenAI, KeyEnv: "OPENAI_COMPATIBLE_2_API_KEY", BaseURLEnv: "OPENAI_COMPATIBLE_2_BASE_URL", ModelsEnv: "OPENAI_COMPATIBLE_2_MODELS", RequiredEnv: []string{"OPENAI_COMPATIBLE_2_BASE_URL", "OPENAI_COMPATIBLE_2_MODELS"}, AutoSyncModels: true},
	{Name: "env-openai-compatible-3", ChannelType: constant.ChannelTypeOpenAI, KeyEnv: "OPENAI_COMPATIBLE_3_API_KEY", BaseURLEnv: "OPENAI_COMPATIBLE_3_BASE_URL", ModelsEnv: "OPENAI_COMPATIBLE_3_MODELS", RequiredEnv: []string{"OPENAI_COMPATIBLE_3_BASE_URL", "OPENAI_COMPATIBLE_3_MODELS"}, AutoSyncModels: true},
}

func buildEnvManagedChannel(definition envChannelDefinition) (*model.Channel, bool) {
	key := strings.TrimSpace(os.Getenv(definition.KeyEnv))
	if key == "" {
		return nil, false
	}
	for _, envName := range definition.RequiredEnv {
		if strings.TrimSpace(os.Getenv(envName)) == "" {
			return nil, false
		}
	}

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(definition.BaseURLEnv)), "/")
	if baseURL == "" && definition.ChannelType < len(constant.ChannelBaseURLs) {
		baseURL = constant.ChannelBaseURLs[definition.ChannelType]
	}
	models := normalizeModelNames(strings.Split(os.Getenv(definition.ModelsEnv), ","))
	if len(models) == 0 {
		models = normalizeModelNames(channelId2Models[definition.ChannelType])
	}

	tag := envManagedChannelTag
	priority := int64(0)
	weight := uint(0)
	autoBan := 1
	channel := &model.Channel{
		Type:        definition.ChannelType,
		Key:         key,
		Status:      common.ChannelStatusEnabled,
		Name:        definition.Name,
		Weight:      &weight,
		CreatedTime: common.GetTimestamp(),
		BaseURL:     &baseURL,
		Models:      strings.Join(models, ","),
		Other:       strings.TrimSpace(os.Getenv(definition.OtherEnv)),
		Group:       "default",
		Priority:    &priority,
		AutoBan:     &autoBan,
		Tag:         &tag,
	}
	channel.SetOtherSettings(dto.ChannelOtherSettings{
		AwsKeyType:                         definition.CredentialMode.AwsKeyType,
		VertexKeyType:                      definition.CredentialMode.VertexKeyType,
		UpstreamModelUpdateCheckEnabled:    definition.AutoSyncModels,
		UpstreamModelUpdateAutoSyncEnabled: definition.AutoSyncModels,
	})
	return channel, true
}

func BootstrapEnvManagedChannels() error {
	if !common.GetEnvOrDefaultBool("ENV_CHANNEL_BOOTSTRAP_ENABLED", false) {
		return nil
	}

	changed := false
	for _, definition := range officialEnvChannelDefinitions {
		desired, configured := buildEnvManagedChannel(definition)
		var existing model.Channel
		err := model.DB.Where("name = ? AND tag = ?", definition.Name, envManagedChannelTag).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if !configured {
			if errors.Is(err, gorm.ErrRecordNotFound) || existing.Status == common.ChannelStatusManuallyDisabled {
				continue
			}
			if err := model.DB.Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&existing).Update("status", common.ChannelStatusManuallyDisabled).Error; err != nil {
					return err
				}
				existing.Status = common.ChannelStatusManuallyDisabled
				return existing.UpdateAbilities(tx)
			}); err != nil {
				return err
			}
			changed = true
			continue
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := model.DB.Transaction(func(tx *gorm.DB) error {
				if err := tx.Create(desired).Error; err != nil {
					return err
				}
				return desired.AddAbilities(tx)
			}); err != nil {
				return err
			}
			changed = true
			continue
		}

		models := mergeModelNames(existing.GetModels(), desired.GetModels())
		settings := existing.GetOtherSettings()
		settings.AwsKeyType = desired.GetOtherSettings().AwsKeyType
		settings.VertexKeyType = desired.GetOtherSettings().VertexKeyType
		settings.UpstreamModelUpdateCheckEnabled = definition.AutoSyncModels
		settings.UpstreamModelUpdateAutoSyncEnabled = definition.AutoSyncModels
		existing.Type = desired.Type
		existing.Key = desired.Key
		existing.Status = common.ChannelStatusEnabled
		existing.BaseURL = desired.BaseURL
		existing.Models = strings.Join(models, ",")
		existing.Other = desired.Other
		existing.SetOtherSettings(settings)
		if err := model.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"type":     existing.Type,
				"key":      existing.Key,
				"status":   existing.Status,
				"base_url": existing.GetBaseURL(),
				"models":   existing.Models,
				"other":    existing.Other,
				"settings": existing.OtherSettings,
			}).Error; err != nil {
				return err
			}
			return existing.UpdateAbilities(tx)
		}); err != nil {
			return err
		}
		changed = true
	}

	if changed {
		refreshChannelRuntimeCache()
		model.InvalidatePricingCache()
	}
	return nil
}
