package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPricingPresetUpdatesFillsMissingValuesOnly(t *testing.T) {
	local := map[string]any{
		"model_ratio":                    map[string]float64{"existing": 1},
		billing_setting.BillingModeField: map[string]string{"existing": "ratio"},
	}
	upstream := map[string]any{
		"model_ratio":                    map[string]any{"existing": 9.0, "new-model": 2.5},
		billing_setting.BillingModeField: map[string]any{"existing": "tiered_expr", "new-model": "tiered_expr"},
	}

	updates, err := buildPricingPresetUpdates(local, upstream, false)
	require.NoError(t, err)
	assert.JSONEq(t, `{"existing":1,"new-model":2.5}`, updates["ModelRatio"])
	assert.JSONEq(t, `{"existing":"ratio","new-model":"tiered_expr"}`, updates["billing_setting.billing_mode"])
}

func TestBuildPricingPresetUpdatesCanOverwriteExistingValues(t *testing.T) {
	local := map[string]any{"model_price": map[string]float64{"image-model": 0.1}}
	upstream := map[string]any{"model_price": map[string]any{"image-model": 0.2}}

	updates, err := buildPricingPresetUpdates(local, upstream, true)
	require.NoError(t, err)
	assert.JSONEq(t, `{"image-model":0.2}`, updates["ModelPrice"])
}

func TestBuildPricingPresetUpdatesIgnoresUnknownFields(t *testing.T) {
	updates, err := buildPricingPresetUpdates(
		map[string]any{},
		map[string]any{"unknown": map[string]any{"model": 1.0}},
		false,
	)
	require.NoError(t, err)
	assert.Empty(t, updates)
}

func TestFetchEnvPricingPresetAcceptsOfficialPayloadWithoutSuccessField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"model_ratio":{"model-a":1.25}}}`))
	}))
	defer server.Close()

	data, err := fetchEnvPricingPreset(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, 1.25, valueMap(data["model_ratio"])["model-a"])
}
