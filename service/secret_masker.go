package service

import (
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/dto"

	"github.com/zricethezav/gitleaks/v8/detect"
)

// secret_masker 基于 gitleaks 内置规则（222+ 条：API key / PEM·SSH 私钥 / JWT / TLS 等）
// 对请求文本做敏感凭证识别与掩码。Detector 构建开销大（编译数百正则），全局单例复用。

var (
	secretDetector     *detect.Detector
	secretDetectorOnce sync.Once
	secretDetectorErr  error
)

func getSecretDetector() (*detect.Detector, error) {
	secretDetectorOnce.Do(func() {
		secretDetector, secretDetectorErr = detect.NewDetectorDefaultConfig()
	})
	return secretDetector, secretDetectorErr
}

// maskValue 掩码单个凭证值：保留头尾少量字符、中间固定替换，不泄露原始长度。
// 风格参照 model.MaskTokenKey。用 rune 切片避免切坏多字节字符。
func maskValue(secret string) string {
	r := []rune(secret)
	n := len(r)
	switch {
	case n <= 4:
		return "***"
	case n <= 12:
		return string(r[:2]) + "***" + string(r[n-2:])
	default:
		return string(r[:4]) + "***" + string(r[n-4:])
	}
}

// MaskText 检测并掩码文本中的敏感凭证。
// 返回：掩码后文本、是否命中、命中的规则 ID 去重列表。
func MaskText(text string) (string, bool, []string) {
	if text == "" {
		return text, false, nil
	}
	d, err := getSecretDetector()
	if err != nil || d == nil {
		return text, false, nil
	}
	findings := d.DetectString(text)
	if len(findings) == 0 {
		return text, false, nil
	}
	masked := text
	rules := make([]string, 0, len(findings))
	seen := make(map[string]struct{})
	for _, f := range findings {
		if f.Secret == "" {
			continue
		}
		masked = strings.ReplaceAll(masked, f.Secret, maskValue(f.Secret))
		if _, ok := seen[f.RuleID]; !ok {
			seen[f.RuleID] = struct{}{}
			rules = append(rules, f.RuleID)
		}
	}
	return masked, masked != text, rules
}

// MaskBytes 对原始字节（JSON / 纯文本）整体检测并掩码，供透传分支兜底使用。
func MaskBytes(body []byte) ([]byte, bool) {
	if len(body) == 0 {
		return body, false
	}
	masked, hit, _ := MaskText(string(body))
	if !hit {
		return body, false
	}
	return []byte(masked), true
}

// sensitiveWalker 由各请求 DTO 实现：遍历自身全部含敏感信息可能的文本片段
// （含工具调用参数、工具结果），用 fn 就地替换。fn 返回 (替换后文本, 是否改动)。
type sensitiveWalker interface {
	WalkSensitiveText(fn func(string) (string, bool)) bool
}

// MaskRequestSecrets 对实现了 WalkSensitiveText 的请求就地脱敏。
// 返回：是否有改动、命中的规则 ID 去重列表（供日志记录）。
//
// 性能：detector.DetectString 每次都要 ToLower 整串 + Aho-Corasick 预过滤 + 跑无 keyword
// 规则，开销固定且不低。为避免对每个文本片段各扫一次（一个多轮带工具调用的请求会有数十段），
// 这里两趟遍历：第一趟把全部片段拼成单缓冲只做 1 次检测；命中后第二趟才对各片段做廉价的
// 字符串替换。无命中（绝大多数请求）仅 1 次扫描。
func MaskRequestSecrets(req dto.Request) (bool, []string) {
	w, ok := req.(sensitiveWalker)
	if !ok {
		return false, nil
	}
	d, err := getSecretDetector()
	if err != nil || d == nil {
		return false, nil
	}

	// 第一趟：收集全部敏感文本片段，拼成单个缓冲一次性送检（fn 不改动）
	var sb strings.Builder
	w.WalkSensitiveText(func(s string) (string, bool) {
		if s != "" {
			sb.WriteString(s)
			sb.WriteByte('\n')
		}
		return s, false
	})
	if sb.Len() == 0 {
		return false, nil
	}
	findings := d.DetectString(sb.String())
	if len(findings) == 0 {
		return false, nil
	}

	// 去重命中的 secret 值与规则 ID
	secrets := make([]string, 0, len(findings))
	secretSeen := make(map[string]struct{}, len(findings))
	rules := make([]string, 0, len(findings))
	ruleSeen := make(map[string]struct{}, len(findings))
	for _, f := range findings {
		if f.Secret != "" {
			if _, ok := secretSeen[f.Secret]; !ok {
				secretSeen[f.Secret] = struct{}{}
				secrets = append(secrets, f.Secret)
			}
		}
		if _, ok := ruleSeen[f.RuleID]; !ok {
			ruleSeen[f.RuleID] = struct{}{}
			rules = append(rules, f.RuleID)
		}
	}
	if len(secrets) == 0 {
		return false, rules
	}

	// 第二趟：对各片段直接做字符串替换，不再跑正则
	changed := w.WalkSensitiveText(func(s string) (string, bool) {
		if s == "" {
			return s, false
		}
		out := s
		for _, sec := range secrets {
			if strings.Contains(out, sec) {
				out = strings.ReplaceAll(out, sec, maskValue(sec))
			}
		}
		if out != s {
			return out, true
		}
		return s, false
	})
	return changed, rules
}
