package openaicompat

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func ShouldChatCompletionsUseResponsesPolicy(policy model_setting.ChatCompletionsToResponsesPolicy, channelID int, channelType int, model string) bool {
	if !policy.IsChannelEnabled(channelID, channelType) {
		return false
	}
	return matchAnyRegex(policy.ModelPatterns, model)
}

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

func ShouldChatCompletionsUseResponsesGlobal(channelID int, channelType int, model string) bool {
	return ShouldChatCompletionsUseResponsesPolicy(
		model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy,
		channelID,
		channelType,
		model,
	)
}
