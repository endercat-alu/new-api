package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newResponsesToClaudeTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	return c, recorder
}

func TestClaudeUsageFromResponsesUsageIncludesCacheCreation(t *testing.T) {
	usage := claudeUsageFromResponsesUsage(&dto.Usage{
		PromptTokens:     100,
		CompletionTokens: 20,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:         30,
			CachedCreationTokens: 10,
		},
	})

	require.NotNil(t, usage)
	require.Equal(t, 100, usage.InputTokens)
	require.Equal(t, 20, usage.OutputTokens)
	require.Equal(t, 30, usage.CacheReadInputTokens)
	require.Equal(t, 10, usage.CacheCreationInputTokens)
	require.NotNil(t, usage.CacheCreation)
	require.Equal(t, 10, usage.CacheCreation.Ephemeral5mInputTokens)
}

func TestOaiResponsesToClaudeHandlerMixedTextAndToolUsesToolStopReason(t *testing.T) {
	c, recorder := newResponsesToClaudeTestContext()
	body := `{
		"id":"resp_1",
		"model":"gpt-4.1",
		"output":[
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"I'll check."}]},
			{"type":"function_call","id":"fc_1","call_id":"call_1","name":"get_weather","arguments":{"city":"Paris"}}
		],
		"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}
	}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	usage, apiErr := OaiResponsesToClaudeHandler(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4.1"}}, resp)

	require.Nil(t, apiErr)
	require.Equal(t, 15, usage.TotalTokens)
	var claudeResp dto.ClaudeResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &claudeResp))
	require.Equal(t, "tool_use", claudeResp.StopReason)
	require.Len(t, claudeResp.Content, 2)
	require.Equal(t, "text", claudeResp.Content[0].Type)
	require.Equal(t, "tool_use", claudeResp.Content[1].Type)
	require.Equal(t, "call_1", claudeResp.Content[1].Id)
}

func TestOaiResponsesToClaudeStreamHandlerKeepsToolAfterText(t *testing.T) {
	oldStreamingTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldStreamingTimeout })

	c, recorder := newResponsesToClaudeTestContext()
	body := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_1","model":"gpt-4.1","created_at":123}}`,
		`data: {"type":"response.output_text.delta","delta":"I'll check."}`,
		`data: {"type":"response.output_item.added","item":{"type":"function_call","id":"fc_1","call_id":"call_1","name":"get_weather","arguments":""}}`,
		`data: {"type":"response.function_call_arguments.delta","item_id":"fc_1","delta":"{\"city\":\"Paris\"}"}`,
		`data: {"type":"response.completed","response":{"id":"resp_1","model":"gpt-4.1","created_at":123,"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}`,
		`data: [DONE]`,
	}, "\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	usage, apiErr := OaiResponsesToClaudeStreamHandler(c, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4.1"}}, resp)

	require.Nil(t, apiErr)
	require.Equal(t, 15, usage.TotalTokens)
	output := recorder.Body.String()
	require.Contains(t, output, `"type":"message_start"`)
	require.Contains(t, output, `"type":"content_block_start"`)
	require.Contains(t, output, `"type":"text_delta"`)
	require.Contains(t, output, `"type":"tool_use"`)
	require.Contains(t, output, `"type":"input_json_delta"`)
	require.Contains(t, output, `"stop_reason":"tool_use"`)
	require.Contains(t, output, `"partial_json":"{\"city\":\"Paris\"}"`)
}
