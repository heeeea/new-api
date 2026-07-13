package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfficialEnvChannelDefinitionsCoverNativeOfficialProviders(t *testing.T) {
	requiredTypes := []int{
		constant.ChannelTypeOpenAI,
		constant.ChannelTypeSora,
		constant.ChannelTypeAnthropic,
		constant.ChannelTypeAzure,
		constant.ChannelTypeGemini,
		constant.ChannelTypeVertexAi,
		constant.ChannelTypeAws,
		constant.ChannelTypeDeepSeek,
		constant.ChannelTypeXai,
		constant.ChannelTypeAli,
		constant.ChannelTypeVolcEngine,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeTencent,
		constant.ChannelTypeMiniMax,
		constant.ChannelTypeMoonshot,
		constant.ChannelTypeZhipu_v4,
		constant.ChannelTypeBaiduV2,
		constant.ChannelTypeXunfei,
		constant.ChannelType360,
		constant.ChannelTypeOpenRouter,
		constant.ChannelTypePerplexity,
		constant.ChannelTypeLingYiWanWu,
		constant.ChannelTypeCohere,
		constant.ChannelTypeJina,
		constant.ChannelCloudflare,
		constant.ChannelTypeSiliconFlow,
		constant.ChannelTypeMistral,
		constant.ChannelTypeCoze,
		constant.ChannelTypeKling,
		constant.ChannelTypeJimeng,
		constant.ChannelTypeVidu,
		constant.ChannelTypeReplicate,
	}

	covered := make(map[int]bool)
	names := make(map[string]bool)
	for _, definition := range officialEnvChannelDefinitions {
		covered[definition.ChannelType] = true
		require.NotEmpty(t, definition.Name)
		require.False(t, names[definition.Name], "duplicate env-managed channel name %s", definition.Name)
		names[definition.Name] = true
		require.NotEmpty(t, definition.KeyEnv)
	}
	for _, channelType := range requiredTypes {
		assert.True(t, covered[channelType], "missing native official channel type %d", channelType)
	}
}

func TestBuildEnvManagedChannelSkipsEmptyKey(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "")
	channel, ok := buildEnvManagedChannel(envChannelDefinition{
		Name:        "env-test",
		ChannelType: constant.ChannelTypeOpenAI,
		KeyEnv:      "TEST_PROVIDER_KEY",
	})
	require.False(t, ok)
	require.Nil(t, channel)
}

func TestBuildEnvManagedChannelSkipsMissingRequiredMetadata(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "secret")
	t.Setenv("TEST_PROVIDER_ACCOUNT_ID", "")
	channel, ok := buildEnvManagedChannel(envChannelDefinition{
		Name:        "env-test",
		ChannelType: constant.ChannelCloudflare,
		KeyEnv:      "TEST_PROVIDER_KEY",
		RequiredEnv: []string{"TEST_PROVIDER_ACCOUNT_ID"},
	})
	require.False(t, ok)
	require.Nil(t, channel)
}

func TestBuildEnvManagedChannelUsesDefaultsAndEnablesModelSync(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "secret")
	channel, ok := buildEnvManagedChannel(envChannelDefinition{
		Name:           "env-openai-test",
		ChannelType:    constant.ChannelTypeOpenAI,
		KeyEnv:         "TEST_PROVIDER_KEY",
		AutoSyncModels: true,
	})
	require.True(t, ok)
	require.NotNil(t, channel)
	assert.Equal(t, "secret", channel.Key)
	assert.Equal(t, common.ChannelStatusEnabled, channel.Status)
	assert.Equal(t, envManagedChannelTag, channel.GetTag())
	assert.Equal(t, constant.ChannelBaseURLs[constant.ChannelTypeOpenAI], channel.GetBaseURL())
	assert.Contains(t, channel.GetModels(), "gpt-4o")
	settings := channel.GetOtherSettings()
	assert.True(t, settings.UpstreamModelUpdateCheckEnabled)
	assert.True(t, settings.UpstreamModelUpdateAutoSyncEnabled)
}

func TestBuildEnvManagedChannelHonorsBaseURLAndModelOverrides(t *testing.T) {
	t.Setenv("TEST_PROVIDER_KEY", "secret")
	t.Setenv("TEST_PROVIDER_BASE_URL", " https://gateway.example/v1/ ")
	t.Setenv("TEST_PROVIDER_MODELS", "model-b, model-a,model-b")
	channel, ok := buildEnvManagedChannel(envChannelDefinition{
		Name:        "env-customized-test",
		ChannelType: constant.ChannelTypeOpenAI,
		KeyEnv:      "TEST_PROVIDER_KEY",
		BaseURLEnv:  "TEST_PROVIDER_BASE_URL",
		ModelsEnv:   "TEST_PROVIDER_MODELS",
	})
	require.True(t, ok)
	assert.Equal(t, "https://gateway.example/v1", channel.GetBaseURL())
	assert.Equal(t, []string{"model-b", "model-a"}, channel.GetModels())
}

func TestBuildEnvManagedChannelSerializesCredentialModes(t *testing.T) {
	t.Setenv("AWS_BEDROCK_CREDENTIALS", "api-key|us-east-1")
	aws, ok := buildEnvManagedChannel(envChannelDefinition{
		Name:        "env-aws-test",
		ChannelType: constant.ChannelTypeAws,
		KeyEnv:      "AWS_BEDROCK_CREDENTIALS",
		CredentialMode: envChannelCredentialMode{
			AwsKeyType: dto.AwsKeyTypeApiKey,
		},
	})
	require.True(t, ok)
	assert.Equal(t, dto.AwsKeyTypeApiKey, aws.GetOtherSettings().AwsKeyType)

	t.Setenv("VERTEX_AI_CREDENTIALS", `{"project_id":"demo"}`)
	vertex, ok := buildEnvManagedChannel(envChannelDefinition{
		Name:        "env-vertex-test",
		ChannelType: constant.ChannelTypeVertexAi,
		KeyEnv:      "VERTEX_AI_CREDENTIALS",
		CredentialMode: envChannelCredentialMode{
			VertexKeyType: dto.VertexKeyTypeJSON,
		},
	})
	require.True(t, ok)
	assert.Equal(t, dto.VertexKeyTypeJSON, vertex.GetOtherSettings().VertexKeyType)
}

func TestBootstrapEnvManagedChannelsCreatesAndUpdatesOneOwnedChannel(t *testing.T) {
	setupModelListControllerTestDB(t)
	t.Setenv("ENV_CHANNEL_BOOTSTRAP_ENABLED", "true")
	t.Setenv("TEST_PROVIDER_KEY", "first-key")
	t.Setenv("TEST_PROVIDER_MODELS", "model-a")
	previousDefinitions := officialEnvChannelDefinitions
	officialEnvChannelDefinitions = []envChannelDefinition{{
		Name:           "env-bootstrap-test",
		ChannelType:    constant.ChannelTypeOpenAI,
		KeyEnv:         "TEST_PROVIDER_KEY",
		ModelsEnv:      "TEST_PROVIDER_MODELS",
		AutoSyncModels: true,
	}}
	t.Cleanup(func() { officialEnvChannelDefinitions = previousDefinitions })

	require.NoError(t, BootstrapEnvManagedChannels())
	var created model.Channel
	require.NoError(t, model.DB.Where("name = ? AND tag = ?", "env-bootstrap-test", envManagedChannelTag).First(&created).Error)
	assert.Equal(t, "first-key", created.Key)
	assert.Equal(t, []string{"model-a"}, created.GetModels())
	var abilityCount int64
	require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id = ?", created.Id).Count(&abilityCount).Error)
	assert.EqualValues(t, 1, abilityCount)

	t.Setenv("TEST_PROVIDER_KEY", "second-key")
	t.Setenv("TEST_PROVIDER_MODELS", "model-b")
	require.NoError(t, BootstrapEnvManagedChannels())
	var channels []model.Channel
	require.NoError(t, model.DB.Where("name = ? AND tag = ?", "env-bootstrap-test", envManagedChannelTag).Find(&channels).Error)
	require.Len(t, channels, 1)
	assert.Equal(t, "second-key", channels[0].Key)
	assert.ElementsMatch(t, []string{"model-a", "model-b"}, channels[0].GetModels())
}

func TestBootstrapEnvManagedChannelsDisablesOwnedChannelWhenKeyIsRemoved(t *testing.T) {
	setupModelListControllerTestDB(t)
	t.Setenv("ENV_CHANNEL_BOOTSTRAP_ENABLED", "true")
	t.Setenv("TEST_PROVIDER_KEY", "configured")
	previousDefinitions := officialEnvChannelDefinitions
	officialEnvChannelDefinitions = []envChannelDefinition{{
		Name:        "env-disable-test",
		ChannelType: constant.ChannelTypeDeepSeek,
		KeyEnv:      "TEST_PROVIDER_KEY",
		ModelsEnv:   "TEST_PROVIDER_MODELS",
	}}
	t.Cleanup(func() { officialEnvChannelDefinitions = previousDefinitions })

	t.Setenv("TEST_PROVIDER_MODELS", "deepseek-chat")
	require.NoError(t, BootstrapEnvManagedChannels())
	t.Setenv("TEST_PROVIDER_KEY", "")
	require.NoError(t, BootstrapEnvManagedChannels())

	var channel model.Channel
	require.NoError(t, model.DB.Where("name = ? AND tag = ?", "env-disable-test", envManagedChannelTag).First(&channel).Error)
	assert.Equal(t, common.ChannelStatusManuallyDisabled, channel.Status)
	var enabledAbilities int64
	require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id = ? AND enabled = ?", channel.Id, true).Count(&enabledAbilities).Error)
	assert.Zero(t, enabledAbilities)
}

func TestBootstrapEnvManagedChannelsPreservesManualChannel(t *testing.T) {
	setupModelListControllerTestDB(t)
	t.Setenv("ENV_CHANNEL_BOOTSTRAP_ENABLED", "true")
	t.Setenv("TEST_PROVIDER_KEY", "env-key")
	previousDefinitions := officialEnvChannelDefinitions
	officialEnvChannelDefinitions = []envChannelDefinition{{
		Name:        "same-name",
		ChannelType: constant.ChannelTypeOpenAI,
		KeyEnv:      "TEST_PROVIDER_KEY",
		ModelsEnv:   "TEST_PROVIDER_MODELS",
	}}
	t.Cleanup(func() { officialEnvChannelDefinitions = previousDefinitions })
	t.Setenv("TEST_PROVIDER_MODELS", "env-model")
	require.NoError(t, model.DB.Create(&model.Channel{
		Name:   "same-name",
		Type:   constant.ChannelTypeOpenAI,
		Key:    "manual-key",
		Models: "manual-model",
		Group:  "default",
		Status: common.ChannelStatusEnabled,
	}).Error)

	require.NoError(t, BootstrapEnvManagedChannels())
	var manual model.Channel
	require.NoError(t, model.DB.Where("name = ? AND (tag IS NULL OR tag <> ?)", "same-name", envManagedChannelTag).First(&manual).Error)
	assert.Equal(t, "manual-key", manual.Key)
	assert.Equal(t, "manual-model", manual.Models)
	var ownedCount int64
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("name = ? AND tag = ?", "same-name", envManagedChannelTag).Count(&ownedCount).Error)
	assert.EqualValues(t, 1, ownedCount)
}
