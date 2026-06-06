package openai

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func usageFromResponsesResponse(resp *dto.OpenAIResponsesResponse) *dto.Usage {
	usage := &dto.Usage{}
	if resp == nil || resp.Usage == nil {
		return usage
	}
	if resp.Usage.InputTokens != 0 {
		usage.PromptTokens = resp.Usage.InputTokens
		usage.InputTokens = resp.Usage.InputTokens
	}
	if resp.Usage.OutputTokens != 0 {
		usage.CompletionTokens = resp.Usage.OutputTokens
		usage.OutputTokens = resp.Usage.OutputTokens
	}
	if resp.Usage.TotalTokens != 0 {
		usage.TotalTokens = resp.Usage.TotalTokens
	} else {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if resp.Usage.InputTokensDetails != nil {
		usage.PromptTokensDetails.CachedTokens = resp.Usage.InputTokensDetails.CachedTokens
		usage.PromptTokensDetails.CachedCreationTokens = resp.Usage.InputTokensDetails.CachedCreationTokens
		usage.PromptTokensDetails.ImageTokens = resp.Usage.InputTokensDetails.ImageTokens
		usage.PromptTokensDetails.AudioTokens = resp.Usage.InputTokensDetails.AudioTokens
	}
	if resp.Usage.CompletionTokenDetails.ReasoningTokens != 0 {
		usage.CompletionTokenDetails.ReasoningTokens = resp.Usage.CompletionTokenDetails.ReasoningTokens
	}
	return usage
}

func claudeUsageFromResponsesUsage(usage *dto.Usage) *dto.ClaudeUsage {
	if usage == nil {
		return nil
	}
	cacheCreation5m, cacheCreation1h := service.NormalizeCacheCreationSplit(
		usage.PromptTokensDetails.CachedCreationTokens,
		usage.ClaudeCacheCreation5mTokens,
		usage.ClaudeCacheCreation1hTokens,
	)
	claudeUsage := &dto.ClaudeUsage{
		InputTokens:                 usage.PromptTokens,
		OutputTokens:                usage.CompletionTokens,
		CacheCreationInputTokens:    usage.PromptTokensDetails.CachedCreationTokens,
		CacheReadInputTokens:        usage.PromptTokensDetails.CachedTokens,
		ClaudeCacheCreation5mTokens: usage.ClaudeCacheCreation5mTokens,
		ClaudeCacheCreation1hTokens: usage.ClaudeCacheCreation1hTokens,
	}
	if cacheCreation5m > 0 || cacheCreation1h > 0 {
		claudeUsage.CacheCreation = &dto.ClaudeCacheCreationUsage{
			Ephemeral5mInputTokens: cacheCreation5m,
			Ephemeral1hInputTokens: cacheCreation1h,
		}
	}
	return claudeUsage
}

func OaiResponsesToClaudeHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	// Check if response body is empty for successful status codes
	if err := service.ValidateResponseBody(resp, body); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeEmptyResponse, http.StatusBadGateway)
	}

	var responsesResp dto.OpenAIResponsesResponse
	if err := common.Unmarshal(body, &responsesResp); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if oaiError := responsesResp.GetOpenAIError(); oaiError != nil && oaiError.Type != "" {
		return nil, types.WithOpenAIError(*oaiError, resp.StatusCode)
	}

	usage := usageFromResponsesResponse(&responsesResp)
	if usage.TotalTokens == 0 {
		usage = service.ResponseText2Usage(c, service.ExtractOutputTextFromResponses(&responsesResp), info.UpstreamModelName, info.GetEstimatePromptTokens())
	}

	contents := make([]dto.ClaudeMediaMessage, 0)
	sawToolCall := false
	for _, out := range responsesResp.Output {
		switch out.Type {
		case "message":
			if out.Role != "" && out.Role != "assistant" {
				continue
			}
			for _, part := range out.Content {
				if part.Text == "" {
					continue
				}
				content := dto.ClaudeMediaMessage{Type: "text"}
				content.SetText(part.Text)
				contents = append(contents, content)
			}
		case "function_call":
			name := strings.TrimSpace(out.Name)
			if name == "" {
				continue
			}
			sawToolCall = true
			callID := strings.TrimSpace(out.CallId)
			if callID == "" {
				callID = strings.TrimSpace(out.ID)
			}
			content := dto.ClaudeMediaMessage{
				Type:  "tool_use",
				Id:    callID,
				Name:  name,
				Input: out.ArgumentsString(),
			}
			var input map[string]any
			if err := common.Unmarshal([]byte(out.ArgumentsString()), &input); err == nil {
				content.Input = input
			}
			contents = append(contents, content)
		}
	}

	stopReason := "end_turn"
	if sawToolCall {
		stopReason = "tool_use"
	}

	claudeResp := &dto.ClaudeResponse{
		Id:         responsesResp.ID,
		Type:       "message",
		Role:       "assistant",
		Model:      responsesResp.Model,
		Content:    contents,
		StopReason: stopReason,
		Usage:      claudeUsageFromResponsesUsage(usage),
	}
	responseBody, err := common.Marshal(claudeResp)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}

	service.IOCopyBytesGracefully(c, resp, responseBody)
	return usage, nil
}

func OaiResponsesToClaudeStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	responseID := helper.GetResponseID(c)
	createAt := time.Now().Unix()
	model := info.UpstreamModelName
	usage := &dto.Usage{}
	var usageText strings.Builder
	var streamErr *types.NewAPIError
	var sentStart bool
	var activeBlockType string
	var activeBlockIndex int
	var nextBlockIndex int
	var sawToolCall bool
	var outputText strings.Builder
	toolCallIndexByID := make(map[string]int)
	toolCallNameByID := make(map[string]string)
	toolCallArgsByID := make(map[string]string)
	toolCallCanonicalIDByItemID := make(map[string]string)

	send := func(resp dto.ClaudeResponse) bool {
		if err := helper.ClaudeData(c, resp); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		return true
	}
	sendStart := func() bool {
		if sentStart {
			return true
		}
		msg := &dto.ClaudeMediaMessage{Id: responseID, Type: "message", Role: "assistant", Model: model, Usage: &dto.ClaudeUsage{InputTokens: info.GetEstimatePromptTokens(), OutputTokens: 0}}
		msg.SetContent(make([]any, 0))
		if !send(dto.ClaudeResponse{Type: "message_start", Message: msg}) {
			return false
		}
		sentStart = true
		return true
	}
	sendStopBlock := func() bool {
		if activeBlockType == "" {
			return true
		}
		idx := activeBlockIndex
		if !send(dto.ClaudeResponse{Type: "content_block_stop", Index: &idx}) {
			return false
		}
		activeBlockType = ""
		return true
	}
	startTextBlock := func(blockType string) bool {
		if !sendStart() || activeBlockType == blockType {
			return activeBlockType == blockType
		}
		if !sendStopBlock() {
			return false
		}
		idx := nextBlockIndex
		nextBlockIndex++
		activeBlockIndex = idx
		activeBlockType = blockType
		content := dto.ClaudeMediaMessage{Type: "text"}
		if blockType == "thinking" {
			content.Type = "thinking"
			content.Thinking = common.GetPointer("")
		} else {
			content.SetText("")
		}
		return send(dto.ClaudeResponse{Type: "content_block_start", Index: &idx, ContentBlock: &content})
	}
	sendTextDelta := func(delta string) bool {
		if delta == "" {
			return true
		}
		if !startTextBlock("text") {
			return false
		}
		idx := activeBlockIndex
		return send(dto.ClaudeResponse{Type: "content_block_delta", Index: &idx, Delta: &dto.ClaudeMediaMessage{Type: "text_delta", Text: common.GetPointer(delta)}})
	}
	sendThinkingDelta := func(delta string) bool {
		if delta == "" {
			return true
		}
		if !startTextBlock("thinking") {
			return false
		}
		idx := activeBlockIndex
		return send(dto.ClaudeResponse{Type: "content_block_delta", Index: &idx, Delta: &dto.ClaudeMediaMessage{Type: "thinking_delta", Thinking: common.GetPointer(delta)}})
	}
	sendToolDelta := func(callID string, name string, argsDelta string) bool {
		callID = strings.TrimSpace(callID)
		if callID == "" {
			return true
		}
		if !sendStart() {
			return false
		}
		sawToolCall = true
		idx, ok := toolCallIndexByID[callID]
		if !ok {
			if !sendStopBlock() {
				return false
			}
			idx = nextBlockIndex
			nextBlockIndex++
			toolCallIndexByID[callID] = idx
			if name != "" {
				toolCallNameByID[callID] = name
			}
			activeBlockIndex = idx
			activeBlockType = "tool_use"
			if !send(dto.ClaudeResponse{Type: "content_block_start", Index: &idx, ContentBlock: &dto.ClaudeMediaMessage{Id: callID, Type: "tool_use", Name: toolCallNameByID[callID], Input: map[string]any{}}}) {
				return false
			}
		} else {
			activeBlockIndex = idx
			activeBlockType = "tool_use"
			if name != "" {
				toolCallNameByID[callID] = name
			}
		}
		if argsDelta == "" {
			return true
		}
		return send(dto.ClaudeResponse{Type: "content_block_delta", Index: &idx, Delta: &dto.ClaudeMediaMessage{Type: "input_json_delta", PartialJson: common.GetPointer(argsDelta)}})
	}

	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {
		if streamErr != nil {
			sr.Stop(streamErr)
			return
		}
		var streamResp dto.ResponsesStreamResponse
		if err := common.UnmarshalJsonStr(data, &streamResp); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
			sr.Stop(streamErr)
			return
		}
		switch streamResp.Type {
		case "response.created":
			if streamResp.Response != nil {
				if streamResp.Response.ID != "" {
					responseID = streamResp.Response.ID
				}
				if streamResp.Response.Model != "" {
					model = streamResp.Response.Model
				}
				if streamResp.Response.CreatedAt != 0 {
					createAt = int64(streamResp.Response.CreatedAt)
				}
			}
		case "response.output_text.delta":
			outputText.WriteString(streamResp.Delta)
			usageText.WriteString(streamResp.Delta)
			if !sendTextDelta(streamResp.Delta) {
				sr.Stop(streamErr)
			}
		case "response.reasoning_summary_text.delta":
			usageText.WriteString(streamResp.Delta)
			if !sendThinkingDelta(streamResp.Delta) {
				sr.Stop(streamErr)
			}
		case "response.output_item.added", "response.output_item.done":
			if streamResp.Item == nil || streamResp.Item.Type != "function_call" {
				return
			}
			itemID := strings.TrimSpace(streamResp.Item.ID)
			callID := strings.TrimSpace(streamResp.Item.CallId)
			if callID == "" {
				callID = itemID
			}
			if itemID != "" && callID != "" {
				toolCallCanonicalIDByItemID[itemID] = callID
			}
			name := strings.TrimSpace(streamResp.Item.Name)
			args := streamResp.Item.ArgumentsString()
			argsDelta := ""
			if args != "" {
				prevArgs := toolCallArgsByID[callID]
				if strings.HasPrefix(args, prevArgs) {
					argsDelta = args[len(prevArgs):]
				} else {
					argsDelta = args
				}
				toolCallArgsByID[callID] = args
			}
			if !sendToolDelta(callID, name, argsDelta) {
				sr.Stop(streamErr)
			}
		case "response.function_call_arguments.delta":
			callID := toolCallCanonicalIDByItemID[strings.TrimSpace(streamResp.ItemID)]
			if callID == "" {
				callID = strings.TrimSpace(streamResp.ItemID)
			}
			toolCallArgsByID[callID] += streamResp.Delta
			if !sendToolDelta(callID, "", streamResp.Delta) {
				sr.Stop(streamErr)
			}
		case "response.completed":
			if streamResp.Response != nil {
				if streamResp.Response.ID != "" {
					responseID = streamResp.Response.ID
				}
				if streamResp.Response.Model != "" {
					model = streamResp.Response.Model
				}
				if streamResp.Response.CreatedAt != 0 {
					createAt = int64(streamResp.Response.CreatedAt)
				}
				usage = usageFromResponsesResponse(streamResp.Response)
			}
		case "response.error", "response.failed":
			if streamResp.Response != nil {
				if oaiErr := streamResp.Response.GetOpenAIError(); oaiErr != nil && oaiErr.Type != "" {
					streamErr = types.WithOpenAIError(*oaiErr, http.StatusInternalServerError)
					sr.Stop(streamErr)
					return
				}
			}
			streamErr = types.NewOpenAIError(fmt.Errorf("responses stream error: %s", streamResp.Type), types.ErrorCodeBadResponse, http.StatusInternalServerError)
			sr.Stop(streamErr)
		}
	})

	if streamErr != nil {
		return nil, streamErr
	}
	if usage.TotalTokens == 0 {
		usage = service.ResponseText2Usage(c, usageText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
	}
	if !sentStart && !sendStart() {
		return nil, streamErr
	}
	if !sendStopBlock() {
		return nil, streamErr
	}
	stopReason := "end_turn"
	if sawToolCall {
		stopReason = "tool_use"
	}
	if !send(dto.ClaudeResponse{Type: "message_delta", Delta: &dto.ClaudeMediaMessage{StopReason: common.GetPointer(stopReason)}, Usage: claudeUsageFromResponsesUsage(usage)}) {
		return nil, streamErr
	}
	if !send(dto.ClaudeResponse{Type: "message_stop"}) {
		return nil, streamErr
	}
	_ = createAt
	return usage, nil
}
