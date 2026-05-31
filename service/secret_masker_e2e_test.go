package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

// 端到端：真实 gitleaks + 真实 DTO，经 MaskRequestSecrets 验证工具字段被掩码。

func TestMaskRequestSecrets_OpenAIToolCalls(t *testing.T) {
	secret := "sk_live_51HqLyjWDarjtT1zdp7dcXa9Kd2Lp7Qw3Zt6Yr1Nb8Mc"
	args := `{"stripe_key":"` + secret + `"}`
	tc, _ := json.Marshal([]map[string]any{
		{"id": "1", "type": "function", "function": map[string]any{"name": "pay", "arguments": args}},
	})
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "assistant", ToolCalls: tc}},
	}
	changed, rules := MaskRequestSecrets(req)
	if !changed {
		t.Fatal("expected changed")
	}
	if len(rules) == 0 {
		t.Error("expected rule ids")
	}
	got := req.Messages[0].ParseToolCalls()[0].Function.Arguments
	if strings.Contains(got, secret) {
		t.Errorf("secret not masked in tool_calls.arguments: %s", got)
	}
}

func TestMaskRequestSecrets_GeminiFunctionResponse(t *testing.T) {
	secret := "sk_live_51HqLyjWDarjtT1zdp7dcXa9Kd2Lp7Qw3Zt6Yr1Nb8Mc"
	req := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Role: "user", Parts: []dto.GeminiPart{{FunctionResponse: &dto.GeminiFunctionResponse{Name: "f", Response: map[string]any{"stripe_key": secret}}}}},
		},
	}
	changed, _ := MaskRequestSecrets(req)
	if !changed {
		t.Fatal("expected changed")
	}
	b, _ := json.Marshal(req.Contents[0].Parts[0].FunctionResponse.Response)
	if strings.Contains(string(b), secret) {
		t.Errorf("secret not masked in functionResponse: %s", b)
	}
}

func TestMaskRequestSecrets_NoFalsePositivePlainText(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "user", Content: "please summarize today's weather"}},
	}
	changed, _ := MaskRequestSecrets(req)
	if changed {
		t.Error("plain text should not be masked")
	}
}
