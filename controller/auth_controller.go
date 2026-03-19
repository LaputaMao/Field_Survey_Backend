package controller

import (
	"Field_Survey_Backend/service"
	"Field_Survey_Backend/utils"
	"net/http"
	// 这里有生成 JWT 的工具

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func LoginHandler(c *gin.Context) {
	var req LoginRequest
	// 参数绑定与基础校验
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误请求无效"})
		return
	}

	// 获取客户端 IP
	clientIP := c.ClientIP()

	// 调用 Service
	user, err := service.Login(req.Username, req.Password, clientIP)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// 生成真实有效的 JWT Token
	token, err := utils.GenerateJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token 生成失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "登录成功",
		"token":   token,
		"user":    user,
	})
}

// VerifyTokenHandler 【新增】VerifyTokenHandler 处理启动验证
func VerifyTokenHandler(c *gin.Context) {
	// 只要能进到这个 Handler，说明已经过了 JWTAuthMiddleware 的校验
	userID, _ := c.Get("userID")
	userRole, _ := c.Get("role")

	// 在这里可以顺便查一下数据库，确保该用户没有被紧急封号（一管/二管操作）
	// 为简单起见，当前只要 Token 有效就放行
	c.JSON(http.StatusOK, gin.H{
		"message": "Token 有效",
		"user_id": userID,
		"role":    userRole,
	})
}
