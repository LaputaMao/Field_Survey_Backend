package middleware

import (
	"net/http"
	"strings"

	"Field_Survey_Backend/utils"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware 验证 Token 的有效性
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 Authorization Header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请求头中缺少授权信息(Token)"})
			c.Abort()
			return
		}

		// 2. 按空格切分 "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token 格式无效，应为 Bearer <token>"})
			c.Abort()
			return
		}

		// 3. 解析 Token
		tokenString := parts[1]
		claims, err := utils.ParseJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token 无效或已过期，请重新登录"})
			c.Abort()
			return
		}

		// 4. 将用户信息写入上下文，方便后续的 Controller 使用
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}
