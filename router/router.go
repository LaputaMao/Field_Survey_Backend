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
	authApi.Use(middleware.JWTAuthMiddleware())
	{
		authApi.GET("/verify-token", controller.VerifyTokenHandler)

		// 3. 只有二管 (sec_admin) 和 三管 (third_admin) 才能访问的管理接口组
		adminApi := authApi.Group("/manage")
		adminApi.Use(middleware.RoleAuthMiddleware("sec_admin", "third_admin"))
		{
			// 导入用户 Excel
			adminApi.POST("/users/import", controller.UploadUsersExcelHandler)
			// 获取我管理的下属用户列表(可带入参 ?name=张三)
			adminApi.GET("/users", controller.GetUsersHandler)
			// 修改某个下属用户信息 (RESTful 风格)
			adminApi.PUT("/users/:id", controller.UpdateUserHandler)
			// 删除某个下属用户 (RESTful 风格)
			adminApi.DELETE("/users/:id", controller.DeleteUserHandler)
		}
	}

	return r
}
