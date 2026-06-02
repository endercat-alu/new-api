package relay

import (
	"bytes"
	"io"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const responsesPromptCacheSessionHeader = "Session_id"

func applyResponsesPromptCacheKey(jsonData []byte, info *relaycommon.RelayInfo) ([]byte, error) {
	sessionID := responsesPromptCacheSessionID(info)
	if sessionID == "" {
		return jsonData, nil
	}

	trimmed := bytes.TrimSpace(jsonData)
	if len(trimmed) == 0 || trimmed[0] != '{' || !gjson.ValidBytes(trimmed) {
		return jsonData, nil
	}
	if gjson.GetBytes(trimmed, "prompt_cache_key").Exists() {
		return jsonData, nil
	}

	return sjson.SetBytes(jsonData, "prompt_cache_key", sessionID)
}

func responsesPassThroughBody(storage common.BodyStorage, info *relaycommon.RelayInfo, hasPromptCacheKey bool) (io.Reader, int64, io.Closer, error) {
	if (hasPromptCacheKey || responsesPromptCacheSessionID(info) == "") && !setting.MaskSecretsEnabled {
		return common.ReaderOnly(storage), 0, nil, nil
	}

	jsonData, err := storage.Bytes()
	if err != nil {
		return nil, 0, nil, err
	}
	if !hasPromptCacheKey {
		jsonData, err = applyResponsesPromptCacheKey(jsonData, info)
		if err != nil {
			return nil, 0, nil, err
		}
	}
	if setting.MaskSecretsEnabled {
		if masked, hit := service.MaskBytes(jsonData); hit {
			jsonData = masked
		}
	}

	body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
	if err != nil {
		return nil, 0, nil, err
	}
	return body, size, closer, nil
}

func responsesPromptCacheSessionID(info *relaycommon.RelayInfo) string {
	if info == nil {
		return ""
	}
	for key, value := range info.RequestHeaders {
		if strings.EqualFold(key, responsesPromptCacheSessionHeader) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
