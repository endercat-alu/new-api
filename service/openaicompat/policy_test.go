package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/stretchr/testify/require"
)

func TestShouldChatCompletionsUseResponsesPolicy(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:       true,
		AllChannels:   false,
		ChannelTypes:  []int{1},
		ModelPatterns: []string{"^gpt-5$"},
	}

	require.True(t, ShouldChatCompletionsUseResponsesPolicy(policy, 10, 1, "gpt-5"))
	require.False(t, ShouldChatCompletionsUseResponsesPolicy(policy, 10, 1, "gpt-4"))
	require.False(t, ShouldChatCompletionsUseResponsesPolicy(policy, 10, 3, "gpt-5"))
}

func TestShouldChatCompletionsUseResponsesChannelMode(t *testing.T) {
	require.True(t, ShouldChatCompletionsUseResponses(
		dto.ChannelOtherSettings{OpenAICompatMode: dto.OpenAICompatModeResponses},
		0,
		0,
		"",
	))
	require.False(t, ShouldChatCompletionsUseResponses(
		dto.ChannelOtherSettings{OpenAICompatMode: dto.OpenAICompatModeChat},
		0,
		0,
		"",
	))
}
