package oaichat

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func ResponseOpenAI2Gemini(openAIResponse *dto.OpenAITextResponse, info *relaycommon.RelayInfo) *dto.GeminiChatResponse {
	totalTokens := openAIResponse.TotalTokens
	if totalTokens == 0 {
		totalTokens = openAIResponse.PromptTokens + openAIResponse.CompletionTokens
	}
	geminiResponse := &dto.GeminiChatResponse{
		Candidates:       make([]dto.GeminiChatCandidate, 0, len(openAIResponse.Choices)),
		HasUsageMetadata: true,
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:     openAIResponse.PromptTokens,
			CandidatesTokenCount: openAIResponse.CompletionTokens,
			TotalTokenCount:      totalTokens,
			BillingUsage:         openAIBillingUsageFromUsage(&openAIResponse.Usage),
		},
	}
	if metadata, ok := geminiBillingMetadataFromOpenAIUsage(&openAIResponse.Usage); ok {
		geminiResponse.UsageMetadata = metadata
	}

	for _, choice := range openAIResponse.Choices {
		candidate := dto.GeminiChatCandidate{
			Index:         int64(choice.Index),
			SafetyRatings: []dto.GeminiChatSafetyRating{},
		}

		var finishReason string
		switch choice.FinishReason {
		case "stop":
			finishReason = "STOP"
		case "length":
			finishReason = "MAX_TOKENS"
		case "content_filter":
			finishReason = "SAFETY"
		case "tool_calls":
			finishReason = "STOP"
		default:
			finishReason = "STOP"
		}
		candidate.FinishReason = &finishReason

		content := dto.GeminiChatContent{
			Role:  "model",
			Parts: make([]dto.GeminiPart, 0),
		}

		textContent := choice.Message.StringContent()
		if textContent != "" {
			part := dto.GeminiPart{
				Text: textContent,
			}
			content.Parts = append(content.Parts, part)
		}

		toolCalls := choice.Message.ParseToolCalls()
		for _, toolCall := range toolCalls {
			var args map[string]interface{}
			if toolCall.Function.Arguments != "" {
				if err := common.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					args = map[string]interface{}{"arguments": toolCall.Function.Arguments}
				}
			} else {
				args = make(map[string]interface{})
			}

			part := dto.GeminiPart{
				FunctionCall: &dto.FunctionCall{
					FunctionName: toolCall.Function.Name,
					Arguments:    args,
				},
			}
			content.Parts = append(content.Parts, part)
		}

		candidate.Content = content
		geminiResponse.Candidates = append(geminiResponse.Candidates, candidate)
	}

	return geminiResponse
}

func StreamResponseOpenAI2Gemini(openAIResponse *dto.ChatCompletionsStreamResponse, info *relaycommon.RelayInfo) *dto.GeminiChatResponse {
	hasContent := false
	hasFinishReason := false
	for _, choice := range openAIResponse.Choices {
		if len(choice.Delta.GetContentString()) > 0 || (choice.Delta.ToolCalls != nil && len(choice.Delta.ToolCalls) > 0) {
			hasContent = true
		}
		if choice.FinishReason != nil {
			hasFinishReason = true
		}
	}

	if !hasContent && !hasFinishReason {
		return nil
	}

	estimatePromptTokens := 0
	if info != nil {
		estimatePromptTokens = info.GetEstimatePromptTokens()
	}
	geminiResponse := &dto.GeminiChatResponse{
		Candidates:       make([]dto.GeminiChatCandidate, 0, len(openAIResponse.Choices)),
		HasUsageMetadata: true,
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:     estimatePromptTokens,
			CandidatesTokenCount: 0, // 流式响应中可能没有完整的 usage 信息
			TotalTokenCount:      estimatePromptTokens,
		},
	}

	if openAIResponse.Usage != nil {
		geminiResponse.UsageMetadata.PromptTokenCount = openAIResponse.Usage.PromptTokens
		geminiResponse.UsageMetadata.CandidatesTokenCount = openAIResponse.Usage.CompletionTokens
		geminiResponse.UsageMetadata.TotalTokenCount = openAIResponse.Usage.TotalTokens
		geminiResponse.UsageMetadata.BillingUsage = openAIBillingUsageFromUsage(openAIResponse.Usage)
		if metadata, ok := geminiBillingMetadataFromOpenAIUsage(openAIResponse.Usage); ok {
			geminiResponse.UsageMetadata = metadata
		}
	}

	for _, choice := range openAIResponse.Choices {
		candidate := dto.GeminiChatCandidate{
			Index:         int64(choice.Index),
			SafetyRatings: []dto.GeminiChatSafetyRating{},
		}

		if choice.FinishReason != nil {
			var finishReason string
			switch *choice.FinishReason {
			case "stop":
				finishReason = "STOP"
			case "length":
				finishReason = "MAX_TOKENS"
			case "content_filter":
				finishReason = "SAFETY"
			case "tool_calls":
				finishReason = "STOP"
			default:
				finishReason = "STOP"
			}
			candidate.FinishReason = &finishReason
		}

		content := dto.GeminiChatContent{
			Role:  "model",
			Parts: make([]dto.GeminiPart, 0),
		}

		if choice.Delta.ToolCalls != nil {
			for _, toolCall := range choice.Delta.ToolCalls {
				var args map[string]interface{}
				if toolCall.Function.Arguments != "" {
					if err := common.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						args = map[string]interface{}{"arguments": toolCall.Function.Arguments}
					}
				} else {
					args = make(map[string]interface{})
				}

				part := dto.GeminiPart{
					FunctionCall: &dto.FunctionCall{
						FunctionName: toolCall.Function.Name,
						Arguments:    args,
					},
				}
				content.Parts = append(content.Parts, part)
			}
		} else {
			textContent := choice.Delta.GetContentString()
			if textContent != "" {
				part := dto.GeminiPart{
					Text: textContent,
				}
				content.Parts = append(content.Parts, part)
			}
		}

		candidate.Content = content
		geminiResponse.Candidates = append(geminiResponse.Candidates, candidate)
	}

	return geminiResponse
}

func geminiBillingMetadataFromOpenAIUsage(usage *dto.Usage) (dto.GeminiUsageMetadata, bool) {
	if usage == nil || usage.BillingUsage == nil || usage.BillingUsage.GeminiUsageMetadata == nil {
		return dto.GeminiUsageMetadata{}, false
	}
	if usage.BillingUsage.Source != dto.BillingUsageSourceGeminiChat && usage.BillingUsage.Semantic != dto.BillingUsageSemanticGemini {
		return dto.GeminiUsageMetadata{}, false
	}
	billingUsage := dto.CloneBillingUsage(usage.BillingUsage)
	if billingUsage == nil || billingUsage.GeminiUsageMetadata == nil {
		return dto.GeminiUsageMetadata{}, false
	}
	return *billingUsage.GeminiUsageMetadata, true
}

func openAIBillingUsageFromUsage(usage *dto.Usage) *dto.BillingUsage {
	if usage == nil {
		return nil
	}
	if existingBillingUsage := dto.CloneBillingUsage(usage.BillingUsage); existingBillingUsage != nil && existingBillingUsage.OpenAIUsage != nil {
		if existingBillingUsage.Source == dto.BillingUsageSourceOAIChat ||
			existingBillingUsage.Source == dto.BillingUsageSourceOAIResponses ||
			existingBillingUsage.Semantic == dto.BillingUsageSemanticOpenAI {
			return existingBillingUsage
		}
	}
	return dto.NewOpenAIChatBillingUsage(usage)
}
