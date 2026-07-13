package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

const defaultEnvPricingPresetURL = "https://basellm.github.io/llm-metadata/api/newapi/ratio_config-v1-base.json"

var pricingPresetOptionKeys = map[string]string{
	"model_ratio":            "ModelRatio",
	"completion_ratio":       "CompletionRatio",
	"cache_ratio":            "CacheRatio",
	"create_cache_ratio":     "CreateCacheRatio",
	"image_ratio":            "ImageRatio",
	"audio_ratio":            "AudioRatio",
	"audio_completion_ratio": "AudioCompletionRatio",
	"model_price":            "ModelPrice",
	"billing_mode":           "billing_setting.billing_mode",
	"billing_expr":           "billing_setting.billing_expr",
}

func buildPricingPresetUpdates(localData, upstreamData map[string]any, overwrite bool) (map[string]string, error) {
	updates := make(map[string]string)
	for _, field := range pricingSyncFields {
		optionKey, ok := pricingPresetOptionKeys[field]
		if !ok {
			continue
		}
		upstreamValues := valueMap(upstreamData[field])
		if len(upstreamValues) == 0 {
			continue
		}

		merged := make(map[string]any)
		for modelName, value := range valueMap(localData[field]) {
			merged[modelName] = normalizeSyncValue(field, value)
		}
		changed := false
		for modelName, value := range upstreamValues {
			if _, exists := merged[modelName]; exists && !overwrite {
				continue
			}
			normalized := normalizeSyncValue(field, value)
			if current, exists := merged[modelName]; exists && valuesEqual(current, normalized) {
				continue
			}
			merged[modelName] = normalized
			changed = true
		}
		if !changed {
			continue
		}
		encoded, err := common.Marshal(merged)
		if err != nil {
			return nil, fmt.Errorf("marshal pricing field %s: %w", field, err)
		}
		updates[optionKey] = string(encoded)
	}
	return updates, nil
}

func fetchEnvPricingPreset(ctx context.Context, sourceURL string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	client := service.GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pricing preset returned HTTP %d", resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, maxRatioConfigBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(body) > maxRatioConfigBytes {
		return nil, fmt.Errorf("pricing preset exceeds %d bytes", maxRatioConfigBytes)
	}
	var payload struct {
		Success *bool          `json:"success"`
		Data    map[string]any `json:"data"`
		Message string         `json:"message"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Success != nil && !*payload.Success {
		return nil, fmt.Errorf("pricing preset rejected request: %s", payload.Message)
	}
	if len(payload.Data) == 0 {
		return nil, fmt.Errorf("pricing preset contains no data")
	}
	return payload.Data, nil
}

func BootstrapEnvPricingPreset() error {
	if !common.GetEnvOrDefaultBool("ENV_PRICING_BOOTSTRAP_ENABLED", false) {
		return nil
	}
	sourceURL := strings.TrimSpace(os.Getenv("ENV_PRICING_BOOTSTRAP_URL"))
	if sourceURL == "" {
		sourceURL = defaultEnvPricingPresetURL
	}
	timeoutSeconds := common.GetEnvOrDefault("ENV_PRICING_BOOTSTRAP_TIMEOUT_SECONDS", 20)
	if timeoutSeconds <= 0 {
		timeoutSeconds = 20
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	upstreamData, err := fetchEnvPricingPreset(ctx, sourceURL)
	if err != nil {
		return err
	}
	updates, err := buildPricingPresetUpdates(
		getLocalPricingSyncData(),
		upstreamData,
		common.GetEnvOrDefaultBool("ENV_PRICING_BOOTSTRAP_OVERWRITE", false),
	)
	if err != nil || len(updates) == 0 {
		return err
	}
	return model.UpdateOptionsBulk(updates)
}
