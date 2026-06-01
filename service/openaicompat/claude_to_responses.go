package openaicompat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func ClaudeRequestToResponsesRequest(req *dto.ClaudeRequest, info *relaycommon.RelayInfo) (*dto.OpenAIResponsesRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	inputItems := make([]map[string]any, 0, len(req.Messages))
	for _, msg := range req.Messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			continue
		}

		if msg.IsStringContent() {
			inputItems = append(inputItems, map[string]any{
				"role":    role,
				"content": msg.GetStringContent(),
			})
			continue
		}

		contents, err := msg.ParseContent()
		if err != nil {
			return nil, err
		}
		contentParts := make([]map[string]any, 0, len(contents))
		for _, part := range contents {
			switch part.Type {
			case "text", "input_text":
				textType := "input_text"
				if role == "assistant" {
					textType = "output_text"
				}
				contentParts = append(contentParts, map[string]any{
					"type": textType,
					"text": part.GetText(),
				})
			case "image":
				if part.Source == nil {
					return nil, fmt.Errorf("image source is required")
				}
				imageURL := part.Source.Url
				if imageURL == "" {
					imageURL = fmt.Sprintf("data:%s;base64,%s", part.Source.MediaType, common.Interface2String(part.Source.Data))
				}
				contentParts = append(contentParts, map[string]any{
					"type":      "input_image",
					"image_url": imageURL,
				})
			case "tool_use":
				inputItems = append(inputItems, map[string]any{
					"type":      "function_call",
					"call_id":   part.Id,
					"name":      part.Name,
					"arguments": toJSONString(part.Input),
				})
			case "tool_result":
				output := part.GetStringContent()
				if output == "" && part.Content != nil {
					b, err := common.Marshal(part.Content)
					if err != nil {
						return nil, err
					}
					output = string(b)
				}
				inputItems = append(inputItems, map[string]any{
					"type":    "function_call_output",
					"call_id": part.ToolUseId,
					"output":  output,
				})
			}
		}
		if len(contentParts) > 0 {
			inputItems = append(inputItems, map[string]any{
				"role":    role,
				"content": contentParts,
			})
		}
	}

	inputRaw, err := common.Marshal(inputItems)
	if err != nil {
		return nil, err
	}

	var instructionsRaw json.RawMessage
	if req.System != nil {
		if req.IsStringSystem() {
			if system := strings.TrimSpace(req.GetStringSystem()); system != "" {
				instructionsRaw, _ = common.Marshal(system)
			}
		} else {
			systems := req.ParseSystem()
			var sb strings.Builder
			for _, system := range systems {
				if text := strings.TrimSpace(system.GetText()); text != "" {
					if sb.Len() > 0 {
						sb.WriteString("\n")
					}
					sb.WriteString(text)
				}
			}
			if sb.Len() > 0 {
				instructionsRaw, _ = common.Marshal(sb.String())
			}
		}
	}

	var toolsRaw json.RawMessage
	if req.Tools != nil {
		tools, _ := common.Any2Type[[]dto.Tool](req.Tools)
		responsesTools := make([]map[string]any, 0, len(tools))
		for _, tool := range tools {
			responsesTools = append(responsesTools, map[string]any{
				"type":        "function",
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			})
		}
		toolsRaw, _ = common.Marshal(responsesTools)
	}

	var toolChoiceRaw json.RawMessage
	if req.ToolChoice != nil {
		choice, _ := common.Any2Type[dto.ClaudeToolChoice](req.ToolChoice)
		switch choice.Type {
		case "auto", "none":
			toolChoiceRaw, _ = common.Marshal(choice.Type)
		case "any":
			toolChoiceRaw, _ = common.Marshal("required")
		case "tool":
			toolChoiceRaw, _ = common.Marshal(map[string]any{"type": "function", "name": choice.Name})
		default:
			toolChoiceRaw, _ = common.Marshal(req.ToolChoice)
		}
	}

	out := &dto.OpenAIResponsesRequest{
		Model:           req.Model,
		Input:           inputRaw,
		Instructions:    instructionsRaw,
		MaxOutputTokens: req.MaxTokens,
		Stream:          req.Stream,
		Temperature:     req.Temperature,
		ToolChoice:      toolChoiceRaw,
		Tools:           toolsRaw,
		TopP:            req.TopP,
		Metadata:        req.Metadata,
	}

	if req.Thinking != nil {
		out.Reasoning = &dto.Reasoning{Summary: "detailed"}
		if req.Thinking.Type == "adaptive" {
			out.Reasoning.Effort = req.GetEfforts()
			if out.Reasoning.Effort == "" {
				out.Reasoning.Effort = "high"
			}
		} else if req.Thinking.Type == "enabled" {
			out.Reasoning.Effort = "high"
		}
	}

	if info != nil && info.SupportStreamOptions && info.IsStream {
		out.StreamOptions = &dto.StreamOptions{IncludeUsage: true}
	}

	return out, nil
}

func toJSONString(v any) string {
	b, err := common.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}
