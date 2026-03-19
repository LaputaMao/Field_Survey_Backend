package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RoleAuthMiddleware 检查用户的角色是否在允许的列表中
func RoleAuthMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 JWT 中间件中获取刚才存入的 role
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无法获取用户角色"})
			c.Abort()
			return
		}

		roleStr := userRole.(string)
		isAllowed := false
		for _, role := range allowedRoles {
			if roleStr == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，禁止访问"})
			c.Abort()
			return
		}

		c.Next()
	}
}
