// Package router router/router.go
package router

import (
	"Field_Survey_Backend/controller"
	"Field_Survey_Backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	//静态服务
	r.Static("/downloads", "./static/shps")

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

		// 1. 二级管理员独有的操作专区
		secAdminApi := authApi.Group("/sec-admin")
		secAdminApi.Use(middleware.RoleAuthMiddleware("sec_admin"))
		{
			// 创建项目
			secAdminApi.POST("/projects", controller.CreateProjectHandler)
			// 分发工作区给三管
			secAdminApi.POST("/workspaces/assign", controller.AssignWorkspaceHandler)
		}

		// 2. 三级管理员独有的操作专区
		thirdAdminApi := authApi.Group("/third-admin")
		thirdAdminApi.Use(middleware.RoleAuthMiddleware("third_admin"))
		{
			// 每次登录或页面刷新、以及通过 JS 定时器轮询调用此接口
			// 可以一次性拉取最新的任务，并依据 unread_count 播放提示音或显示弹窗红点
			thirdAdminApi.GET("/my-workspaces", controller.GetMyWorkspacesHandler)
			// 三管点进任务详情去下载 shp 时，触发此接口消除已读状态
			thirdAdminApi.PUT("/workspaces/:id/read", controller.ReadWorkspaceHandler)
		}
	}

	return r
}
