// Middleware - 限流

package gateway

import (
	"github.com/gin-gonic/gin"
)

// RateLimitMiddleware 速率限制中间件
// ponytail: 配额检查暂未启用，接用户充值 token 系统后恢复
func (g *DefaultGateway) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
