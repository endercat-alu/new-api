package dto

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
)

// sensitive_walk 为各请求格式实现 WalkSensitiveText：遍历自身全部可能含敏感凭证的
// 文本片段（含工具调用参数、工具结果），用 fn 就地替换。fn 返回 (替换后文本, 是否命中)。
// 返回是否有任何改动。由 service.MaskRequestSecrets 通过 duck-typing 调用。
//
// 设计要点：
//   - 纯字符串字段直接调 fn；
//   - any / map / json.RawMessage 等结构化字段（工具参数/结果）先序列化为 JSON 文本再检测，
//     命中后反序列化写回（保留 key 名上下文，更利于规则命中）；反序列化失败则保留原值兜底。

type maskFn = func(string) (string, bool)

// maskAny 处理 any 字段：string 直接检测；其余序列化为 JSON 检测后回写。
func maskAny(v any, fn maskFn) (any, bool) {
	if v == nil {
		return v, false
	}
	if s, ok := v.(string); ok {
		if m, hit := fn(s); hit {
			return m, true
		}
		return v, false
	}
	b, err := common.Marshal(v)
	if err != nil {
		return v, false
	}
	m, hit := fn(string(b))
	if !hit {
		return v, false
	}
	var nv any
	if err := common.Unmarshal([]byte(m), &nv); err != nil {
		return v, false // 掩码破坏了结构，保留原值
	}
	return nv, true
}

// maskRaw 处理 json.RawMessage：整体检测，命中且仍为合法 JSON 才回写。
func maskRaw(raw json.RawMessage, fn maskFn) (json.RawMessage, bool) {
	if len(raw) == 0 {
		return raw, false
	}
	m, hit := fn(string(raw))
	if !hit {
		return raw, false
	}
	if !json.Valid([]byte(m)) {
		return raw, false
	}
	return json.RawMessage(m), true
}

// ─────────────────── OpenAI Chat Completions（旧格式）───────────────────

func (r *GeneralOpenAIRequest) WalkSensitiveText(fn maskFn) bool {
	changed := false
	maskStrAny := func(v any) any {
		switch t := v.(type) {
		case string:
			if m, hit := fn(t); hit {
				changed = true
				return m
			}
		case []any:
			for i, item := range t {
				if s, ok := item.(string); ok {
					if m, hit := fn(s); hit {
						t[i] = m
						changed = true
					}
				}
			}
		}
		return v
	}

	r.Prompt = maskStrAny(r.Prompt)
	r.Input = maskStrAny(r.Input)

	for i := range r.Messages {
		msg := &r.Messages[i]
		// content
		if msg.Content != nil {
			if msg.IsStringContent() {
				if m, hit := fn(msg.StringContent()); hit {
					msg.SetStringContent(m)
					changed = true
				}
			} else {
				parts := msg.ParseContent()
				sub := false
				for j := range parts {
					if parts[j].Type == ContentTypeText && parts[j].Text != "" {
						if m, hit := fn(parts[j].Text); hit {
							parts[j].Text = m
							sub = true
						}
					}
				}
				if sub {
					msg.SetMediaContent(parts)
					changed = true
				}
			}
		}
		// tool_calls.arguments（现有计费抽取遗漏，重点补）
		if len(msg.ToolCalls) > 0 {
			tcs := msg.ParseToolCalls()
			sub := false
			for k := range tcs {
				if tcs[k].Function.Arguments != "" {
					if m, hit := fn(tcs[k].Function.Arguments); hit {
						tcs[k].Function.Arguments = m
						sub = true
					}
				}
			}
			if sub {
				msg.SetToolCalls(tcs)
				changed = true
			}
		}
	}
	return changed
}

// ─────────────────── OpenAI Responses（新格式）───────────────────

func (r *OpenAIResponsesRequest) WalkSensitiveText(fn maskFn) bool {
	changed := false
	if len(r.Input) > 0 {
		var items []map[string]any
		if err := common.Unmarshal(r.Input, &items); err == nil {
			for _, item := range items {
				switch item["type"] {
				case "function_call": // arguments（重点补）
					if nv, hit := maskAny(item["arguments"], fn); hit {
						item["arguments"] = nv
						changed = true
					}
				case "function_call_output": // output（重点补）
					if nv, hit := maskAny(item["output"], fn); hit {
						item["output"] = nv
						changed = true
					}
				default: // message / 其它：content
					if nv, hit := maskAny(item["content"], fn); hit {
						item["content"] = nv
						changed = true
					}
				}
			}
			if changed {
				if b, err := common.Marshal(items); err == nil {
					r.Input = b
				}
			}
		} else if nv, hit := maskRaw(r.Input, fn); hit { // 纯字符串 input
			r.Input = nv
			changed = true
		}
	}
	if nv, hit := maskRaw(r.Instructions, fn); hit {
		r.Instructions = nv
		changed = true
	}
	if nv, hit := maskRaw(r.Prompt, fn); hit {
		r.Prompt = nv
		changed = true
	}
	return changed
}

// ─────────────────── Claude ───────────────────

func (c *ClaudeRequest) WalkSensitiveText(fn maskFn) bool {
	changed := false
	// system
	if c.System != nil {
		if c.IsStringSystem() {
			if m, hit := fn(c.GetStringSystem()); hit {
				c.SetStringSystem(m)
				changed = true
			}
		} else {
			media := c.ParseSystem()
			if walkClaudeMedia(media, fn) {
				c.System = media
				changed = true
			}
		}
	}
	// messages
	for i := range c.Messages {
		msg := &c.Messages[i]
		if msg.IsStringContent() {
			if m, hit := fn(msg.GetStringContent()); hit {
				msg.SetStringContent(m)
				changed = true
			}
			continue
		}
		media, err := msg.ParseContent()
		if err != nil {
			continue
		}
		if walkClaudeMedia(media, fn) {
			msg.SetContent(media)
			changed = true
		}
	}
	return changed
}

// walkClaudeMedia 处理 Claude content block 切片，返回是否有改动。
func walkClaudeMedia(media []ClaudeMediaMessage, fn maskFn) bool {
	changed := false
	for j := range media {
		m := &media[j]
		switch m.Type {
		case "text":
			if t := m.GetText(); t != "" {
				if masked, hit := fn(t); hit {
					m.SetText(masked)
					changed = true
				}
			}
		case "tool_use": // input（参数）
			if nv, hit := maskAny(m.Input, fn); hit {
				m.Input = nv
				changed = true
			}
		case "tool_result": // content（结果）
			if nv, hit := maskAny(m.Content, fn); hit {
				m.Content = nv
				changed = true
			}
		}
	}
	return changed
}

// ─────────────────── Gemini ───────────────────

func (r *GeminiChatRequest) WalkSensitiveText(fn maskFn) bool {
	changed := false
	for i := range r.Contents {
		if walkGeminiContent(&r.Contents[i], fn) {
			changed = true
		}
	}
	if r.SystemInstructions != nil {
		if walkGeminiContent(r.SystemInstructions, fn) {
			changed = true
		}
	}
	for i := range r.Requests { // 批量请求
		if r.Requests[i].WalkSensitiveText(fn) {
			changed = true
		}
	}
	return changed
}

func walkGeminiContent(c *GeminiChatContent, fn maskFn) bool {
	changed := false
	for i := range c.Parts {
		p := &c.Parts[i]
		if p.Text != "" {
			if m, hit := fn(p.Text); hit {
				p.Text = m
				changed = true
			}
		}
		if p.FunctionCall != nil { // args（参数，重点补）
			if nv, hit := maskAny(p.FunctionCall.Arguments, fn); hit {
				p.FunctionCall.Arguments = nv
				changed = true
			}
		}
		if p.FunctionResponse != nil && p.FunctionResponse.Response != nil { // response（结果，重点补）
			if nv, hit := maskAny(p.FunctionResponse.Response, fn); hit {
				if m, ok := nv.(map[string]any); ok {
					p.FunctionResponse.Response = m
					changed = true
				}
			}
		}
		if p.ExecutableCode != nil && p.ExecutableCode.Code != "" {
			if m, hit := fn(p.ExecutableCode.Code); hit {
				p.ExecutableCode.Code = m
				changed = true
			}
		}
		if p.CodeExecutionResult != nil && p.CodeExecutionResult.Output != "" {
			if m, hit := fn(p.CodeExecutionResult.Output); hit {
				p.CodeExecutionResult.Output = m
				changed = true
			}
		}
	}
	return changed
}

// ─────────────────── Embedding / Rerank / Image ───────────────────

func (r *EmbeddingRequest) WalkSensitiveText(fn maskFn) bool {
	changed := false
	switch v := r.Input.(type) {
	case string:
		if m, hit := fn(v); hit {
			r.Input = m
			changed = true
		}
	case []any:
		for i, item := range v {
			if s, ok := item.(string); ok {
				if m, hit := fn(s); hit {
					v[i] = m
					changed = true
				}
			}
		}
	}
	return changed
}

func (r *RerankRequest) WalkSensitiveText(fn maskFn) bool {
	changed := false
	if r.Query != "" {
		if m, hit := fn(r.Query); hit {
			r.Query = m
			changed = true
		}
	}
	for i, doc := range r.Documents {
		if s, ok := doc.(string); ok {
			if m, hit := fn(s); hit {
				r.Documents[i] = m
				changed = true
			}
		} else if nv, hit := maskAny(doc, fn); hit {
			r.Documents[i] = nv
			changed = true
		}
	}
	return changed
}

func (i *ImageRequest) WalkSensitiveText(fn maskFn) bool {
	if i.Prompt == "" {
		return false
	}
	if m, hit := fn(i.Prompt); hit {
		i.Prompt = m
		return true
	}
	return false
}
