package service

import (
	"strings"
	"testing"
)

func TestMaskText_DetectsSecrets(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		secret string // 应被掩码移除的子串
	}{
		{
			name:   "private key block",
			input:  "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDX9k2nQh7Zt6Yr\n1Nb8Mc4Hf5Gj0Vs2DqXa9Kd2Lp7Qw3Zt6Yr1Nb8Mc4Hf5Gj0Vs2DqkLmNoPqRsT\n-----END PRIVATE KEY-----",
			secret: "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDX9k2nQh7Zt6Yr",
		},
		{
			name:   "github token",
			input:  "export GITHUB_TOKEN=ghp_Xa9Kd2Lp7Qw3Zt6Yr1Nb8Mc4Hf5Gj0Vs2Dq",
			secret: "ghp_Xa9Kd2Lp7Qw3Zt6Yr1Nb8Mc4Hf5Gj0Vs2Dq",
		},
		{
			name:   "stripe token",
			input:  "stripe sk_live_51HqLyjWDarjtT1zdp7dcXa9Kd2Lp7Qw3Zt6Yr1Nb8Mc",
			secret: "sk_live_51HqLyjWDarjtT1zdp7dcXa9Kd2Lp7Qw3Zt6Yr1Nb8Mc",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			masked, hit, rules := MaskText(tc.input)
			if !hit {
				t.Fatalf("expected hit, got none; rules=%v", rules)
			}
			if strings.Contains(masked, tc.secret) {
				t.Errorf("secret not masked: still contains %q\nmasked=%q", tc.secret, masked)
			}
			if len(rules) == 0 {
				t.Errorf("expected rule ids, got none")
			}
		})
	}
}

func TestMaskText_NoSecret(t *testing.T) {
	in := "just a normal sentence describing the weather today"
	masked, hit, rules := MaskText(in)
	if hit || masked != in || len(rules) != 0 {
		t.Errorf("expected no detection, got hit=%v masked=%q rules=%v", hit, masked, rules)
	}
}

func TestMaskText_Empty(t *testing.T) {
	masked, hit, _ := MaskText("")
	if hit || masked != "" {
		t.Errorf("empty should be no-op, got hit=%v masked=%q", hit, masked)
	}
}

func TestMaskValue(t *testing.T) {
	cases := map[string]string{
		"ab":                  "***",
		"abcd":                "***",
		"abcdefgh":            "ab***gh",
		"sk-1234567890abcdef": "sk-1***cdef",
	}
	for in, want := range cases {
		if got := maskValue(in); got != want {
			t.Errorf("maskValue(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestMaskBytes(t *testing.T) {
	// JSON 形态（贴近 tool_calls.arguments），stripe 前缀规则稳定命中
	secret := "sk_live_51HqLyjWDarjtT1zdp7dcXa9Kd2Lp7Qw3Zt6Yr1Nb8Mc"
	out, hit := MaskBytes([]byte(`{"stripe_key":"` + secret + `"}`))
	if !hit {
		t.Fatalf("expected hit")
	}
	if strings.Contains(string(out), secret) {
		t.Errorf("secret not masked in bytes: %s", out)
	}
}
