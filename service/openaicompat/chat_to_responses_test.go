package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestChatCompletionsRequestToResponsesRequestPreservesToolStrict(t *testing.T) {
	strict := false
	req := &dto.GeneralOpenAIRequest{
		Model: "gpt-5",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
		Tools: []dto.ToolCallRequest{
			{
				Type: "function",
				Function: dto.FunctionRequest{
					Name:        "get_weather",
					Description: "Get weather",
					Parameters:  map[string]any{"type": "object"},
					Strict:      &strict,
				},
			},
		},
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)

	var tools []map[string]any
	require.NoError(t, common.Unmarshal(out.Tools, &tools))
	require.Len(t, tools, 1)
	require.Equal(t, strict, tools[0]["strict"])
}

func TestChatCompletionsRequestToResponsesRequestRaisesReasoningMaxTokensFloor(t *testing.T) {
	maxTokens := uint(16)
	req := &dto.GeneralOpenAIRequest{
		Model:           "gpt-5",
		ReasoningEffort: "high",
		MaxTokens:       &maxTokens,
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.NotNil(t, out.MaxOutputTokens)
	require.Equal(t, uint(1024), *out.MaxOutputTokens)
}

func TestChatCompletionsRequestToResponsesRequestPreservesPromptCacheKey(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Model:          "gpt-5",
		PromptCacheKey: "explicit-key",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	}

	out, err := ChatCompletionsRequestToResponsesRequest(req)
	require.NoError(t, err)
	require.Equal(t, "explicit-key", common.JsonRawMessageToString(out.PromptCacheKey))
}
