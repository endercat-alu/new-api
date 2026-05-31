package dto

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

// mockMask 把含 "SECRET" 的文本替换为 "MASKED"，用于验证遍历是否覆盖各字段（不依赖 gitleaks）。
func mockMask(s string) (string, bool) {
	if strings.Contains(s, "SECRET") {
		return strings.ReplaceAll(s, "SECRET", "MASKED"), true
	}
	return s, false
}

// OpenAI Chat：tool_calls.arguments（现有计费抽取遗漏点，必须覆盖）
func TestWalk_OpenAIChat_ToolCallsArguments(t *testing.T) {
	r := &GeneralOpenAIRequest{
		Messages: []Message{
			{Role: "assistant", ToolCalls: json.RawMessage(`[{"id":"1","type":"function","function":{"name":"f","arguments":"{\"key\":\"SECRET\"}"}}]`)},
		},
	}
	if !r.WalkSensitiveText(mockMask) {
		t.Fatal("expected changed")
	}
	tcs := r.Messages[0].ParseToolCalls()
	if strings.Contains(tcs[0].Function.Arguments, "SECRET") {
		t.Errorf("tool_calls.arguments not masked: %s", tcs[0].Function.Arguments)
	}
}

func TestWalk_OpenAIChat_StringContent(t *testing.T) {
	r := &GeneralOpenAIRequest{
		Messages: []Message{{Role: "tool", Content: "result is SECRET value"}},
	}
	if !r.WalkSensitiveText(mockMask) {
		t.Fatal("expected changed")
	}
	if strings.Contains(r.Messages[0].StringContent(), "SECRET") {
		t.Errorf("content not masked")
	}
}

// Gemini：functionCall.args + functionResponse.response（遗漏点）
func TestWalk_Gemini_FunctionCallResponse(t *testing.T) {
	r := &GeminiChatRequest{
		Contents: []GeminiChatContent{
			{Role: "model", Parts: []GeminiPart{{FunctionCall: &FunctionCall{FunctionName: "f", Arguments: map[string]any{"key": "SECRET"}}}}},
			{Role: "user", Parts: []GeminiPart{{FunctionResponse: &GeminiFunctionResponse{Name: "f", Response: map[string]any{"out": "SECRET"}}}}},
		},
	}
	if !r.WalkSensitiveText(mockMask) {
		t.Fatal("expected changed")
	}
	b0, _ := common.Marshal(r.Contents[0].Parts[0].FunctionCall.Arguments)
	if strings.Contains(string(b0), "SECRET") {
		t.Errorf("functionCall.args not masked: %s", b0)
	}
	b1, _ := common.Marshal(r.Contents[1].Parts[0].FunctionResponse.Response)
	if strings.Contains(string(b1), "SECRET") {
		t.Errorf("functionResponse not masked: %s", b1)
	}
}

// Claude：tool_use.input + tool_result.content
func TestWalk_Claude_ToolUseResult(t *testing.T) {
	r := &ClaudeRequest{
		Messages: []ClaudeMessage{
			{Role: "assistant", Content: []any{map[string]any{"type": "tool_use", "name": "f", "input": map[string]any{"key": "SECRET"}}}},
			{Role: "user", Content: []any{map[string]any{"type": "tool_result", "content": "SECRET output"}}},
		},
	}
	if !r.WalkSensitiveText(mockMask) {
		t.Fatal("expected changed")
	}
	b, _ := common.Marshal(r.Messages)
	if strings.Contains(string(b), "SECRET") {
		t.Errorf("claude tool data not masked: %s", b)
	}
}

// Responses：function_call.arguments + function_call_output.output
func TestWalk_Responses_ToolItems(t *testing.T) {
	r := &OpenAIResponsesRequest{
		Input: json.RawMessage(`[{"type":"function_call","name":"f","arguments":"{\"key\":\"SECRET\"}"},{"type":"function_call_output","output":"SECRET out"}]`),
	}
	if !r.WalkSensitiveText(mockMask) {
		t.Fatal("expected changed")
	}
	if strings.Contains(string(r.Input), "SECRET") {
		t.Errorf("responses tool data not masked: %s", r.Input)
	}
}

func TestWalk_SimpleFormats(t *testing.T) {
	e := &EmbeddingRequest{Input: "embed SECRET"}
	if !e.WalkSensitiveText(mockMask) || strings.Contains(e.Input.(string), "SECRET") {
		t.Errorf("embedding not masked: %v", e.Input)
	}

	rr := &RerankRequest{Query: "q SECRET", Documents: []any{"doc SECRET"}}
	if !rr.WalkSensitiveText(mockMask) {
		t.Fatal("rerank expected changed")
	}
	if strings.Contains(rr.Query, "SECRET") || strings.Contains(rr.Documents[0].(string), "SECRET") {
		t.Errorf("rerank not masked")
	}

	im := &ImageRequest{Prompt: "draw SECRET"}
	if !im.WalkSensitiveText(mockMask) || strings.Contains(im.Prompt, "SECRET") {
		t.Errorf("image not masked")
	}
}
