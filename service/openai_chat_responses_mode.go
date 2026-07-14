package service

import (
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service/openaicompat"
	"github.com/QuantumNous/new-api/service/relayconvert"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	return relayconvert.ShouldChatCompletionsUseResponsesPolicy(policy, channelID, channelType, model)
}

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return relayconvert.ShouldChatCompletionsUseResponsesGlobal(channelID, channelType, model)
}

// ShouldChatCompletionsUseResponses applies per-channel openai_compat_mode, then falls back to global policy.
func ShouldChatCompletionsUseResponses(channelSettings dto.ChannelOtherSettings, channelID int, channelType int, model string) bool {
	switch channelSettings.OpenAICompatMode {
	case dto.OpenAICompatModeResponses:
		return true
	case dto.OpenAICompatModeChat:
		return false
	default:
		return ShouldChatCompletionsUseResponsesGlobal(channelID, channelType, model)
	}
}

func ClaudeRequestToResponsesRequest(request *dto.ClaudeRequest, info *relaycommon.RelayInfo) (*dto.OpenAIResponsesRequest, error) {
	return openaicompat.ClaudeRequestToResponsesRequest(request, info)
}
