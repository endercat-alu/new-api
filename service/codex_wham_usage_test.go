package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchCodexWhamUsage_SetsHeadersAndPath(t *testing.T) {
	var got *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	status, body, err := FetchCodexWhamUsage(
		context.Background(),
		server.Client(),
		server.URL+"/",
		" token ",
		" account ",
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"ok":true}`, string(body))
	require.NotNil(t, got)
	assert.Equal(t, http.MethodGet, got.Method)
	assert.Equal(t, "/backend-api/wham/usage", got.URL.Path)
	assert.Equal(t, "Bearer token", got.Header.Get("Authorization"))
	assert.Equal(t, "account", got.Header.Get("chatgpt-account-id"))
	assert.Equal(t, "application/json", got.Header.Get("Accept"))
	assert.Equal(t, "codex_cli_rs", got.Header.Get("originator"))
}

func TestFetchCodexWhamRateLimitResetCredits_Path(t *testing.T) {
	var got *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"credits":[]}`))
	}))
	t.Cleanup(server.Close)

	status, body, err := FetchCodexWhamRateLimitResetCredits(
		context.Background(),
		server.Client(),
		server.URL,
		"token",
		"account",
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"credits":[]}`, string(body))
	require.NotNil(t, got)
	assert.Equal(t, "/backend-api/wham/rate-limit-reset-credits", got.URL.Path)
}

func TestConsumeCodexWhamRateLimitResetCredit_PostsRedeemRequestID(t *testing.T) {
	var got *http.Request
	var payload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &payload))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"windows_reset":2}`))
	}))
	t.Cleanup(server.Close)

	status, body, err := ConsumeCodexWhamRateLimitResetCredit(
		context.Background(),
		server.Client(),
		server.URL,
		"token",
		"account",
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"windows_reset":2}`, string(body))
	require.NotNil(t, got)
	assert.Equal(t, http.MethodPost, got.Method)
	assert.Equal(t, "/backend-api/wham/rate-limit-reset-credits/consume", got.URL.Path)
	assert.Equal(t, "application/json", got.Header.Get("Content-Type"))
	require.NotEmpty(t, payload["redeem_request_id"])
}

func TestCodexWhamRequest_ValidationErrors(t *testing.T) {
	_, _, err := FetchCodexWhamUsage(context.Background(), nil, "https://example.com", "t", "a")
	require.Error(t, err)

	client := &http.Client{}
	_, _, err = FetchCodexWhamUsage(context.Background(), client, "", "t", "a")
	require.Error(t, err)
	_, _, err = FetchCodexWhamUsage(context.Background(), client, "https://example.com", "", "a")
	require.Error(t, err)
	_, _, err = FetchCodexWhamUsage(context.Background(), client, "https://example.com", "t", "")
	require.Error(t, err)
}

func TestCodexWhamRequest_PropagatesUpstreamStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"auth"}`))
	}))
	t.Cleanup(server.Close)

	status, body, err := FetchCodexWhamUsage(
		context.Background(),
		server.Client(),
		server.URL,
		"token",
		"account",
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.JSONEq(t, `{"error":"auth"}`, string(body))
}
