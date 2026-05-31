package setting

// MaskSecretsEnabled 是否对入站请求内容识别并掩码敏感凭证（API key / 私钥 / TLS 证书等）后再转发上游。默认关闭，opt-in。
//
// 该功能与敏感词过滤（sensitive.go 中的 CheckSensitive* / SensitiveWords）相互独立、互不依赖：
// 凭证脱敏保护用户提供的密钥不外泄给上游，敏感词过滤拦截违规内容。两者各自有独立开关与执行路径。
var MaskSecretsEnabled = false
