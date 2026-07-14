package channel

import (
	"net/http"
	"strings"
)

const (
	DefaultUpstreamUserAgent = "Mozilla/5.0 (compatible; OpenAI-API-Client/1.0)"
	DefaultCodexUserAgent    = "codex_cli_rs/0.50.0 (Linux 6.8.0; x86_64) terminal"
	DefaultCodexOriginator   = "codex_cli_rs"
)

func resolveUpstreamUserAgent(clientUserAgent string) string {
	if ua := strings.TrimSpace(clientUserAgent); ua != "" {
		return ua
	}
	return DefaultUpstreamUserAgent
}

func applyDefaultUpstreamUserAgent(req *http.Header, clientUserAgent string) {
	if req == nil {
		return
	}
	if strings.TrimSpace(req.Get("User-Agent")) != "" {
		return
	}
	req.Set("User-Agent", resolveUpstreamUserAgent(clientUserAgent))
}

func ensureRequestUserAgent(req *http.Request) {
	if req == nil {
		return
	}
	if strings.TrimSpace(req.Header.Get("User-Agent")) != "" {
		return
	}
	req.Header.Set("User-Agent", DefaultUpstreamUserAgent)
}
