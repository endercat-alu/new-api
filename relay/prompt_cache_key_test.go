package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestApplyResponsesPromptCacheKeyKeepsExplicitKey(t *testing.T) {
	jsonData := []byte(`{"model":"gpt-5","prompt_cache_key":"explicit-key"}`)
	info := &relaycommon.RelayInfo{RequestHeaders: map[string]string{"Session_id": "sess-123"}}

	out, err := applyResponsesPromptCacheKey(jsonData, info)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(out, &body))
	require.Equal(t, "explicit-key", body["prompt_cache_key"])
}

func TestApplyResponsesPromptCacheKeyUsesSessionID(t *testing.T) {
	jsonData := []byte(`{"model":"gpt-5"}`)
	info := &relaycommon.RelayInfo{RequestHeaders: map[string]string{"Session_id": "sess-123"}}

	out, err := applyResponsesPromptCacheKey(jsonData, info)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(out, &body))
	require.Equal(t, "sess-123", body["prompt_cache_key"])
}

func TestApplyResponsesPromptCacheKeyKeepsEmptyWhenNoSessionID(t *testing.T) {
	jsonData := []byte(`{"model":"gpt-5"}`)
	info := &relaycommon.RelayInfo{RequestHeaders: map[string]string{"Request-Id": "req-123"}}

	out, err := applyResponsesPromptCacheKey(jsonData, info)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(out, &body))
	require.NotContains(t, body, "prompt_cache_key")
}
