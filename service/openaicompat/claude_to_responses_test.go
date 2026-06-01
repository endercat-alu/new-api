package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestClaudeRequestToResponsesRequestPreservesToolStrict(t *testing.T) {
	for _, strict := range []bool{true, false} {
		req := &dto.ClaudeRequest{
			Model: "claude-sonnet-4-5",
			Messages: []dto.ClaudeMessage{
				{Role: "user", Content: "hello"},
			},
			Tools: []dto.Tool{
				{
					Name:        "get_weather",
					Description: "Get weather",
					InputSchema: map[string]interface{}{"type": "object"},
					Strict:      &strict,
				},
			},
		}

		out, err := ClaudeRequestToResponsesRequest(req, nil)
		require.NoError(t, err)

		var tools []map[string]any
		require.NoError(t, common.Unmarshal(out.Tools, &tools))
		require.Len(t, tools, 1)
		require.Equal(t, strict, tools[0]["strict"])
	}
}

func TestClaudeRequestToResponsesRequestNormalizesSmallMaxTokens(t *testing.T) {
	maxTokens := uint(1)
	req := &dto.ClaudeRequest{
		Model:     "claude-sonnet-4-5",
		MaxTokens: &maxTokens,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}

	out, err := ClaudeRequestToResponsesRequest(req, nil)
	require.NoError(t, err)
	require.NotNil(t, out.MaxOutputTokens)
	require.Equal(t, uint(16), *out.MaxOutputTokens)
}

func TestClaudeRequestToResponsesRequestRaisesThinkingMaxTokensFloor(t *testing.T) {
	maxTokens := uint(16)
	req := &dto.ClaudeRequest{
		Model:     "claude-sonnet-4-5",
		MaxTokens: &maxTokens,
		Thinking:  &dto.Thinking{Type: "adaptive"},
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}

	out, err := ClaudeRequestToResponsesRequest(req, nil)
	require.NoError(t, err)
	require.NotNil(t, out.MaxOutputTokens)
	require.Equal(t, uint(1024), *out.MaxOutputTokens)
}

func TestClaudeRequestToResponsesRequestDoesNotMapStopSequencesToTruncation(t *testing.T) {
	req := &dto.ClaudeRequest{
		Model:         "claude-sonnet-4-5",
		StopSequences: []string{"STOP"},
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}

	out, err := ClaudeRequestToResponsesRequest(req, nil)

	require.NoError(t, err)
	require.Empty(t, out.Truncation)
}

func TestClaudeRequestToResponsesRequestRequiresImageSource(t *testing.T) {
	req := &dto.ClaudeRequest{
		Model: "claude-sonnet-4-5",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "image"},
				},
			},
		},
	}

	_, err := ClaudeRequestToResponsesRequest(req, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "image source is required")
}

func TestClaudeRequestToResponsesRequestStringifiesToolResultOutput(t *testing.T) {
	textPart := dto.ClaudeMediaMessage{Type: "text"}
	textPart.SetText("tool result")
	req := &dto.ClaudeRequest{
		Model: "claude-sonnet-4-5",
		Messages: []dto.ClaudeMessage{
			{
				Role: "user",
				Content: []dto.ClaudeMediaMessage{
					{Type: "tool_result", ToolUseId: "call_1", Content: []dto.ClaudeMediaMessage{textPart}},
				},
			},
		},
	}

	out, err := ClaudeRequestToResponsesRequest(req, nil)
	require.NoError(t, err)

	var items []map[string]any
	require.NoError(t, common.Unmarshal(out.Input, &items))
	require.Len(t, items, 1)
	require.Equal(t, "function_call_output", items[0]["type"])
	require.Equal(t, "call_1", items[0]["call_id"])
	output, ok := items[0]["output"].(string)
	require.True(t, ok)
	require.Contains(t, output, "tool result")
}

func TestClaudeRequestToResponsesRequestMapsTopLevelCacheControlRetention(t *testing.T) {
	cacheControl, err := common.Marshal(map[string]any{"type": "ephemeral"})
	require.NoError(t, err)
	req := &dto.ClaudeRequest{
		Model:        "claude-sonnet-4-5",
		CacheControl: cacheControl,
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}

	out, err := ClaudeRequestToResponsesRequest(req, nil)
	require.NoError(t, err)
	require.Equal(t, "in_memory", responsesRetention(t, out.PromptCacheRetention))
	require.Empty(t, out.PromptCacheKey)
}

func TestClaudeRequestToResponsesRequestMapsSystemCacheControlOneHourRetention(t *testing.T) {
	cacheControl, err := common.Marshal(map[string]any{"type": "ephemeral", "ttl": "1h"})
	require.NoError(t, err)
	system := dto.ClaudeMediaMessage{Type: "text", CacheControl: cacheControl}
	system.SetText("stable system")
	req := &dto.ClaudeRequest{
		Model:  "claude-sonnet-4-5",
		System: []dto.ClaudeMediaMessage{system},
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}

	out, err := ClaudeRequestToResponsesRequest(req, nil)
	require.NoError(t, err)
	require.Equal(t, "24h", responsesRetention(t, out.PromptCacheRetention))
	require.Empty(t, out.PromptCacheKey)
}

func TestClaudeRequestToResponsesRequestMapsMessageCacheControlRetention(t *testing.T) {
	cacheControl, err := common.Marshal(map[string]any{"type": "ephemeral"})
	require.NoError(t, err)
	part := dto.ClaudeMediaMessage{Type: "text", CacheControl: cacheControl}
	part.SetText("stable prefix")
	req := &dto.ClaudeRequest{
		Model: "claude-sonnet-4-5",
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: []dto.ClaudeMediaMessage{part}},
		},
	}

	out, err := ClaudeRequestToResponsesRequest(req, nil)
	require.NoError(t, err)
	require.Equal(t, "in_memory", responsesRetention(t, out.PromptCacheRetention))
	require.Empty(t, out.PromptCacheKey)
}

func TestClaudeRequestToResponsesRequestDoesNotSetPromptCacheFieldsWithoutCacheControl(t *testing.T) {
	req := &dto.ClaudeRequest{
		Model: "claude-sonnet-4-5",
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "hello"},
		},
	}

	out, err := ClaudeRequestToResponsesRequest(req, nil)
	require.NoError(t, err)
	require.Empty(t, out.PromptCacheRetention)
	require.Empty(t, out.PromptCacheKey)
}

func responsesRetention(t *testing.T, raw []byte) string {
	t.Helper()
	var retention string
	require.NoError(t, common.Unmarshal(raw, &retention))
	return retention
}
