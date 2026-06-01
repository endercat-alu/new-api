package relay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/gin-gonic/gin"
)

type responsesCacheDebugPayload struct {
	Source               string   `json:"source,omitempty"`
	Model                string   `json:"model,omitempty"`
	PromptCacheKey       string   `json:"prompt_cache_key,omitempty"`
	PromptCacheRetention string   `json:"prompt_cache_retention,omitempty"`
	InputHash            string   `json:"input_hash,omitempty"`
	InstructionsHash     string   `json:"instructions_hash,omitempty"`
	ToolsHash            string   `json:"tools_hash,omitempty"`
	ToolsCount           int      `json:"tools_count,omitempty"`
	ToolNames            []string `json:"tool_names,omitempty"`
	ToolStrictValues     []string `json:"tool_strict_values,omitempty"`
	ToolChoiceHash       string   `json:"tool_choice_hash,omitempty"`
	ParallelToolCalls    string   `json:"parallel_tool_calls,omitempty"`
	MetadataHash         string   `json:"metadata_hash,omitempty"`
	FullBodyHash         string   `json:"full_body_hash,omitempty"`
}

type responsesCacheDebugRequest struct {
	Model                string          `json:"model"`
	Input                json.RawMessage `json:"input"`
	Instructions         json.RawMessage `json:"instructions"`
	Tools                json.RawMessage `json:"tools"`
	ToolChoice           json.RawMessage `json:"tool_choice"`
	ParallelToolCalls    json.RawMessage `json:"parallel_tool_calls"`
	Metadata             json.RawMessage `json:"metadata"`
	PromptCacheKey       json.RawMessage `json:"prompt_cache_key"`
	PromptCacheRetention json.RawMessage `json:"prompt_cache_retention"`
}

func logResponsesCacheDebug(c *gin.Context, source string, jsonData []byte) {
	if !common.DebugEnabled {
		return
	}

	var req responsesCacheDebugRequest
	if err := common.Unmarshal(jsonData, &req); err != nil {
		logger.LogDebug(c, "responses cache debug source=%s body_hash=%s parse_error=%s", source, hashJSON(jsonData), err.Error())
		return
	}

	toolNames, toolStrictValues, toolsCount := responsesToolsDebug(req.Tools)
	payload := responsesCacheDebugPayload{
		Source:               source,
		Model:                req.Model,
		PromptCacheKey:       common.JsonRawMessageToString(req.PromptCacheKey),
		PromptCacheRetention: common.JsonRawMessageToString(req.PromptCacheRetention),
		InputHash:            hashJSON(req.Input),
		InstructionsHash:     hashJSON(req.Instructions),
		ToolsHash:            hashJSON(req.Tools),
		ToolsCount:           toolsCount,
		ToolNames:            toolNames,
		ToolStrictValues:     toolStrictValues,
		ToolChoiceHash:       hashJSON(req.ToolChoice),
		ParallelToolCalls:    common.JsonRawMessageToString(req.ParallelToolCalls),
		MetadataHash:         hashJSON(req.Metadata),
		FullBodyHash:         hashJSON(jsonData),
	}

	data, err := common.Marshal(payload)
	if err != nil {
		logger.LogDebug(c, "responses cache debug source=%s marshal_error=%s", source, err.Error())
		return
	}
	logger.LogDebug(c, "responses cache debug: %s", data)
}

func hashJSON(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ""
	}

	var v any
	if err := common.Unmarshal(trimmed, &v); err == nil {
		if canonical, err := common.Marshal(v); err == nil {
			trimmed = canonical
		}
	}

	sum := sha256.Sum256(trimmed)
	return hex.EncodeToString(sum[:])
}

func responsesToolsDebug(raw []byte) ([]string, []string, int) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil, 0
	}

	var tools []map[string]any
	if err := common.Unmarshal(raw, &tools); err != nil {
		return nil, nil, 0
	}

	names := make([]string, 0, len(tools))
	strictValues := make([]string, 0, len(tools))
	for _, tool := range tools {
		if name, ok := tool["name"].(string); ok && name != "" {
			names = append(names, name)
		} else if fn, ok := tool["function"].(map[string]any); ok {
			if name, ok := fn["name"].(string); ok && name != "" {
				names = append(names, name)
			}
		}

		if strict, exists := tool["strict"]; exists {
			strictValues = append(strictValues, fmt.Sprintf("%v", strict))
		} else if fn, ok := tool["function"].(map[string]any); ok {
			if strict, exists := fn["strict"]; exists {
				strictValues = append(strictValues, fmt.Sprintf("%v", strict))
			} else {
				strictValues = append(strictValues, "missing")
			}
		} else {
			strictValues = append(strictValues, "missing")
		}
	}

	return names, strictValues, len(tools)
}
