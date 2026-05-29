package middleware

import (
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	// SecureVerificationSessionKey 安全验证的 session key（与 controller 保持一致）
	SecureVerificationSessionKey       = "secure_verified_at"
	secureVerificationMethodSessionKey = "secure_verified_method"
	// SecureVerificationTimeout 验证有效期（秒）
	SecureVerificationTimeout = 300 // 5分钟
)

// SecureVerificationRequired 安全验证中间件
// 检查用户是否在有效时间内通过了安全验证
// 如果未验证或验证已过期，返回 401 错误
func SecureVerificationRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if checkSecureVerification(c, true) {
			c.Next()
		}
	}
}

func ChannelKeyVerificationRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.ChannelKeyVerificationEnabled {
			c.Next()
			return
		}
		if checkSecureVerification(c, true) {
			c.Next()
		}
	}
}

func checkSecureVerification(c *gin.Context, abortOnFailure bool) bool {
	userId := c.GetInt("id")
	if userId == 0 {
		if abortOnFailure {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "未登录",
			})
			c.Abort()
		}
		return false
	}

	session := sessions.Default(c)
	verifiedAtRaw := session.Get(SecureVerificationSessionKey)

	if verifiedAtRaw == nil {
		if abortOnFailure {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "需要安全验证",
				"code":    "VERIFICATION_REQUIRED",
			})
			c.Abort()
		}
		return false
	}

	verifiedAt, ok := verifiedAtRaw.(int64)
	if !ok {
		clearSecureVerificationSession(session)
		if abortOnFailure {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "验证状态异常，请重新验证",
				"code":    "VERIFICATION_INVALID",
			})
			c.Abort()
		}
		return false
	}

	elapsed := time.Now().Unix() - verifiedAt
	if elapsed >= SecureVerificationTimeout {
		clearSecureVerificationSession(session)
		if abortOnFailure {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": "验证已过期，请重新验证",
				"code":    "VERIFICATION_EXPIRED",
			})
			c.Abort()
		}
		return false
	}

	return true
}

func clearSecureVerificationSession(session sessions.Session) {
	session.Delete(SecureVerificationSessionKey)
	session.Delete(secureVerificationMethodSessionKey)
	_ = session.Save()
}

// OptionalSecureVerification 可选的安全验证中间件
// 如果用户已验证，则在 context 中设置标记，但不阻止请求继续
// 用于某些需要区分是否已验证的场景
func OptionalSecureVerification() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkSecureVerification(c, false) {
			c.Set("secure_verified", false)
			c.Next()
			return
		}

		session := sessions.Default(c)
		verifiedAt, _ := session.Get(SecureVerificationSessionKey).(int64)
		c.Set("secure_verified", true)
		c.Set("secure_verified_at", verifiedAt)
		c.Next()
	}
}

// ClearSecureVerification 清除安全验证状态
// 用于用户登出或需要强制重新验证的场景
func ClearSecureVerification(c *gin.Context) {
	session := sessions.Default(c)
	clearSecureVerificationSession(session)
}
