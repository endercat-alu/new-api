package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenameToolRewritesRequestAndResponse(t *testing.T) {
	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"mode": "rename_tool",
						"from": "web_search",
						"to":   "search_web",
					},
				},
			},
		},
	}

	request := []byte(`{
		"tools":[{"type":"function","function":{"name":"web_search"}}],
		"tool_choice":{"type":"function","function":{"name":"web_search"}},
		"messages":[{"role":"assistant","tool_calls":[{"function":{"name":"web_search"}}]}]
	}`)
	got, err := ApplyParamOverrideWithRelayInfo(request, info)
	require.NoError(t, err)
	assertJSONEqual(t, `{
		"tools":[{"type":"function","function":{"name":"search_web"}}],
		"tool_choice":{"type":"function","function":{"name":"search_web"}},
		"messages":[{"role":"assistant","tool_calls":[{"function":{"name":"search_web"}}]}]
	}`, string(got))
	require.Equal(t, []ToolNameRewrite{{ClientName: "web_search", UpstreamName: "search_web"}}, info.ToolNameRewrites)

	response := []byte(`{
		"choices":[{"message":{"tool_calls":[{"function":{"name":"search_web","arguments":"{\"query\":\"search_web\"}"}}]}}]
	}`)
	got, err = RewriteOpenAIChatResponseToolNames(response, info)
	require.NoError(t, err)
	assertJSONEqual(t, `{
		"choices":[{"message":{"tool_calls":[{"function":{"name":"web_search","arguments":"{\"query\":\"search_web\"}"}}]}}]
	}`, string(got))
}

func TestRenameToolRewritesStreamingResponse(t *testing.T) {
	info := &RelayInfo{
		ToolNameRewrites: []ToolNameRewrite{{ClientName: "web_search", UpstreamName: "search_web"}},
	}

	response := []byte(`{"choices":[{"delta":{"tool_calls":[{"function":{"name":"search_web"}}]}}]}`)
	got, err := RewriteOpenAIChatResponseToolNames(response, info)
	require.NoError(t, err)
	assertJSONEqual(t, `{"choices":[{"delta":{"tool_calls":[{"function":{"name":"web_search"}}]}}]}`, string(got))
}

func TestRenameToolRejectsReverseMappingCollision(t *testing.T) {
	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{"mode": "rename_tool", "from": "first", "to": "shared"},
					map[string]interface{}{"mode": "rename_tool", "from": "second", "to": "shared"},
				},
			},
		},
	}

	_, err := ApplyParamOverrideWithRelayInfo([]byte(`{"tools":[{"function":{"name":"first"}},{"function":{"name":"second"}}]}`), info)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mapped from both")
}
