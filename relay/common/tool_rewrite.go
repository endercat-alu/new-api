package common

import (
	"fmt"
	"strings"

	rootcommon "github.com/QuantumNous/new-api/common"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const paramOverrideToolNameRecorder = "__param_override_tool_name_recorder"

type toolNameRewriteRecorder struct {
	rewrites         []ToolNameRewrite
	clientToUpstream map[string]string
	upstreamToClient map[string]string
}

func newToolNameRewriteRecorder() *toolNameRewriteRecorder {
	return &toolNameRewriteRecorder{
		clientToUpstream: make(map[string]string),
		upstreamToClient: make(map[string]string),
	}
}

func (r *toolNameRewriteRecorder) record(clientName, upstreamName string) error {
	clientName = strings.TrimSpace(clientName)
	upstreamName = strings.TrimSpace(upstreamName)
	if clientName == "" || upstreamName == "" || clientName == upstreamName {
		return nil
	}

	if existing, ok := r.clientToUpstream[clientName]; ok && existing != upstreamName {
		return fmt.Errorf("tool %q is mapped to both %q and %q", clientName, existing, upstreamName)
	}
	if existing, ok := r.upstreamToClient[upstreamName]; ok && existing != clientName {
		return fmt.Errorf("upstream tool %q is mapped from both %q and %q", upstreamName, existing, clientName)
	}
	if _, ok := r.clientToUpstream[clientName]; ok {
		return nil
	}

	r.clientToUpstream[clientName] = upstreamName
	r.upstreamToClient[upstreamName] = clientName
	r.rewrites = append(r.rewrites, ToolNameRewrite{
		ClientName:   clientName,
		UpstreamName: upstreamName,
	})
	return nil
}

func (r *toolNameRewriteRecorder) appendTo(info *RelayInfo) {
	if r == nil || info == nil {
		return
	}
	for _, rewrite := range r.rewrites {
		alreadyPresent := false
		for _, existing := range info.ToolNameRewrites {
			if existing == rewrite {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			info.ToolNameRewrites = append(info.ToolNameRewrites, rewrite)
		}
	}
}

var requestToolNamePaths = []string{
	"tools.*.function.name",
	"tools.*.name",
	"tool_choice.function.name",
	"tool_choice.name",
	"tool_choice.*.name",
	"function_call.name",
	"messages.*.tool_calls.*.function.name",
	"messages.*.function_call.name",
}

var openAIChatResponseToolNamePaths = []string{
	"choices.*.message.tool_calls.*.function.name",
	"choices.*.message.function_call.name",
	"choices.*.delta.tool_calls.*.function.name",
	"choices.*.delta.function_call.name",
}

func rewriteRequestToolNames(data []byte, clientName, upstreamName string) ([]byte, bool, error) {
	return rewriteToolNamesAtPaths(data, requestToolNamePaths, clientName, upstreamName)
}

// RewriteOpenAIChatResponseToolNames rewrites only OpenAI Chat Completions tool
// call names. Tool arguments and ordinary text are left untouched.
func RewriteOpenAIChatResponseToolNames(data []byte, info *RelayInfo) ([]byte, error) {
	if info == nil || len(info.ToolNameRewrites) == 0 || len(data) == 0 {
		return data, nil
	}

	result := data
	for _, rewrite := range info.ToolNameRewrites {
		var err error
		result, _, err = rewriteToolNamesAtPaths(result, openAIChatResponseToolNamePaths, rewrite.UpstreamName, rewrite.ClientName)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func rewriteToolNamesAtPaths(data []byte, patterns []string, from, to string) ([]byte, bool, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" || from == to {
		return data, false, nil
	}

	var root interface{}
	if err := rootcommon.Unmarshal(data, &root); err != nil {
		return nil, false, err
	}

	paths := make([]string, 0)
	seen := make(map[string]struct{})
	for _, pattern := range patterns {
		var resolved []string
		if strings.Contains(pattern, "*") {
			resolved = collectWildcardPaths(root, strings.Split(pattern, "."), nil)
		} else {
			resolved = []string{pattern}
		}
		for _, path := range resolved {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}

	result := data
	changed := false
	for _, path := range paths {
		current := gjson.GetBytes(result, path)
		if current.Type != gjson.String || current.String() != from {
			continue
		}
		next, err := sjson.SetBytes(result, path, to)
		if err != nil {
			return nil, false, err
		}
		result = next
		changed = true
	}
	return result, changed, nil
}
