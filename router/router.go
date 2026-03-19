// Package router router/router.go
package router

import (
	"Field_Survey_Backend/controller"
	"Field_Survey_Backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	api := r.Group("/api/v1")
	{
		api.POST("/login", controller.LoginHandler)
	}

	// 需要验证 Token 的路由组
	authApi := r.Group("/api/v1")
	authApi.Use(middleware.JWTAuthMiddleware()) // 挂载 JWT 中间件
	{
		// App 启动时的验证接口
		authApi.GET("/verify-token", controller.VerifyTokenHandler)
		// 后续你的所有需要权限的接口都写在这里，比如：
		// authApi.POST("/data/upload", controller.UploadDataHandler)
	}

	return r
}
