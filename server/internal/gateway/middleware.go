// Middleware — 认证与限流

package gateway

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 用户认证中间件
func (g *DefaultGateway) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从请求头获取 token
		token := c.GetHeader("Authorization")
		if token == "" {
			// 允许匿名访问（部分功能限制）
			c.Set("user_id", "anonymous")
			c.Set("user_tier", "free")
			c.Next()
			return
		}

		// 2. 解析 token（简化版，实际应接 JWT）
		userID, err := parseToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// 3. 获取用户信息（简化版，实际应查数据库）
		c.Set("user_id", userID)
		c.Set("user_tier", "free") // 默认免费用户
		c.Next()
	}
}

// RateLimitMiddleware 速率限制中间件
func (g *DefaultGateway) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			userID = "anonymous"
		}

		// TODO: 接用户充值 token 系统后恢复配额检查
		// status, err := g.usageTracker.CheckQuota(userID, c.GetString("user_tier"))
		// if err != nil {
		// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "quota check failed"})
		// 	c.Abort()
		// 	return
		// }
		// if !status.Allowed {
		// 	c.JSON(http.StatusTooManyRequests, gin.H{
		// 		"error":     "quota exceeded",
		// 		"reason":    status.Reason,
		// 		"remaining": status.Remaining,
		// 	})
		// 	c.Abort()
		// 	return
		// }
		// c.Header("X-Quota-Remaining", strconv.Itoa(status.Remaining))
		// c.Header("X-Quota-Limit", strconv.Itoa(status.LimitToday))

		c.Next()
	}
}

// parseToken 解析 token（简化版占位）
func parseToken(token string) (string, error) {
	// 实际实现：JWT 验证
	// 目前直接返回 token 作为 userID（开发模式）
	if len(token) > 7 && token[:7] == "Bearer " {
		return token[7:], nil
	}
	return token, nil
}
