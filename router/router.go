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
		adminApi.Use(middleware.RoleAuthMiddleware("sec_admin", "third_admin", "first_admin"))
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

		// 0. 一级管理员独有的操作专区
		firstAdminApi := authApi.Group("/first-admin")
		firstAdminApi.Use(middleware.RoleAuthMiddleware("first_admin"))
		{
			// 导入项目表
			firstAdminApi.POST("/projects/import", controller.ImportProjectsHandler)
			// 导入人员表
			firstAdminApi.POST("/users/import", controller.ImportUsersHandler)
			// 合并导入（项目和人员）
			firstAdminApi.POST("/import-all", controller.CombinedImportHandler)

			// 展示接口
			firstAdminApi.GET("/projects-tree", controller.GetFirstAdminProjectsTreeHandler)
			firstAdminApi.GET("/workspace-dashboard", controller.GetFirstAdminWorkspaceDashboardHandler)
			firstAdminApi.GET("/points", controller.GetFirstAdminPointsHandler)
			firstAdminApi.GET("/points/:id/properties", controller.GetFirstAdminPointPropertiesHandler)
			// 导出接口
			//firstAdminApi.GET("/workspaces/:workspace_id/export", controller.ExportWorkspaceDataHandler)
			//firstAdminApi.GET("/workspaces/:workspace_id/export/trajectories", controller.ExportWorkspaceTrajectoriesHandler)
			//firstAdminApi.GET("/workspaces/:workspace_id/export/points", controller.ExportWorkspacePointsHandler)
			// 新增：全国底图基建
			firstAdminApi.POST("/basic-shps/upload", controller.UploadBasicShpHandler)
			firstAdminApi.GET("/basic-shps", controller.GetBasicShpListHandler)
			firstAdminApi.GET("/basic-shps/geojson", controller.GetBasicShpGeoJSONHandler)

			firstAdminApi.GET("/workspace/:id/export-shp", controller.ExportWorkspaceShpHandler)

		}

		// 1. 二级管理员独有的操作专区
		secAdminApi := authApi.Group("/sec-admin")
		secAdminApi.Use(middleware.RoleAuthMiddleware("sec_admin"))
		{
			//// 创建项目
			secAdminApi.POST("/projects", controller.CreateProjectHandler)
			//// 分发工作区给三管
			//secAdminApi.POST("/workspaces/assign", controller.AssignWorkspaceHandler)

			// 新增：全国底图基建
			//secAdminApi.POST("/basic-shps/upload", controller.UploadBasicShpHandler)
			//secAdminApi.GET("/basic-shps", controller.GetBasicShpListHandler)
			//secAdminApi.GET("/basic-shps/geojson", controller.GetBasicShpGeoJSONHandler)

			// 重写：裁切实分发
			secAdminApi.POST("/workspaces/assign", controller.AssignWorkspaceHandler)

			// 展示接口
			secAdminApi.GET("/projects-tree", controller.GetSecAdminProjectsTreeHandler)
			secAdminApi.GET("/workspace-dashboard", controller.GetSecAdminWorkspaceDashboardHandler)
			secAdminApi.GET("/points", controller.GetSecAdminPointsHandler)
			secAdminApi.GET("/points/:id/properties", controller.GetSecAdminPointPropertiesHandler)
			// 新增：获取项目下属工作区GeoJSON
			secAdminApi.GET("/workspaces/geojson", controller.GetSecAdminWorkspaceGeoJSONHandler)
			// 导出接口
			//secAdminApi.GET("/workspaces/:workspace_id/export", controller.ExportWorkspaceDataHandler)
			//secAdminApi.GET("/workspaces/:workspace_id/export/trajectories", controller.ExportWorkspaceTrajectoriesHandler)
			//secAdminApi.GET("/workspaces/:workspace_id/export/points", controller.ExportWorkspacePointsHandler)
			secAdminApi.GET("/workspace/:id/export-shp", controller.ExportWorkspaceShpHandler)
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

			// 派发任务给调查员
			// tip 使用 arcGIS 按照'姓名'字段切分成多个 shp 文件,名称不需要固定命名方式
			thirdAdminApi.POST("/tasks/assign", controller.BulkAssignTaskHandler)

			// Web端左侧边栏：拿到管理结构
			thirdAdminApi.GET("/web/workspaces-tree", controller.GetWebWorkspaceTreeHandler)
			// Web端中间地图大屏：拿到大JSON
			thirdAdminApi.GET("/web/task-dashboard", controller.GetWebTaskDetailHandler)
			thirdAdminApi.GET("/workspace-dashboard", controller.GetThirdAdminWorkspaceDashboardHandler)

			// 台账报表分页查询: GET /api/v1/third-admin/points?page=1&username=张三&date=2024-03-12
			thirdAdminApi.GET("/points", controller.GetPointListHandler)

			// 获取点位表单明细: GET /api/v1/third-admin/points/42/properties
			thirdAdminApi.GET("/points/:id/properties", controller.GetPointPropertiesHandler)
			// 导出接口
			//thirdAdminApi.GET("/workspaces/:workspace_id/export", controller.ExportWorkspaceDataHandler)
			//thirdAdminApi.GET("/workspaces/:workspace_id/export/trajectories", controller.ExportWorkspaceTrajectoriesHandler)
			//thirdAdminApi.GET("/workspaces/:workspace_id/export/points", controller.ExportWorkspacePointsHandler)
			// ⭐ 新增：工作区空间数据集一键导出
			// 用法：GET /api/v1/third-admin/workspace/1/export-shp
			thirdAdminApi.GET("/workspace/:id/export-shp", controller.ExportWorkspaceShpHandler)
			// 新增：获取工作区GeoJSON
			thirdAdminApi.GET("/workspaces/geojson", controller.GetThirdAdminWorkspaceGeoJSONHandler)
		}

		// --- 【调查员App业务组】 ---
		userApi := authApi.Group("/user")
		userApi.Use(middleware.RoleAuthMiddleware("user"))
		{
			// 1. App轮询或刷新用的获取任务列表 API
			userApi.GET("/my-tasks", controller.GetSurveyorTasksHandler)

			// 2. App点击该任务时调用，消除红点并直接返回地图可以渲染的 GeoJSON 格式！
			// tip 暂时弃用
			userApi.GET("/tasks/:id/geojson", controller.ReadTaskAndGetGeoJSONHandler)

			// ⭐ 升级后的接口：包含了规划数据(点+线)和实际轨迹数据
			userApi.GET("/tasks/:id/detail", controller.ReadTaskDetailHandler)

			// ⭐ 新增：调查员获取自己某一个点位的完整表单与照片
			userApi.GET("/points/:id/properties", controller.GetSurveyorPointPropertiesHandler)

			//一键完成任务
			userApi.PUT("/tasks/:id/complete", controller.CompleteTaskHandler)

			// 3.上传真实巡护轨迹
			userApi.POST("/routes/upload", controller.UploadActualRouteHandler)

			// 4. 自动填表
			// tip 注意新增 shp 或者 tiff 数据时,需要使用shp2pgsql -s 4326 -I my_file.shp table_a | psql -d my_db
			// tip 和raster2pgsql -s 4326 -I -C -M -t 100x100 "E:\Field_survey\precipitation.tif" public.precipitation | psql  -U postgres -d field_survey_db
			// tip 命令进行底表导入,然后在注册表中增加对应 数据源 和 SQL 语句
			userApi.POST("/auto-fill", controller.AutoFillAttrHandler)

			userApi.GET("/points/next-number", controller.GetNextPointNumberHandler)

			// 5.上传调查点
			userApi.POST("/points/upload", controller.UploadPointHandler)

			// ⭐ 新增：混合修改接口点位属性 (使用 POST 以更好支持 Multipart 请求)
			userApi.POST("/points/:id/update", controller.UpdatePointHandler)

			// 考虑到前端需要显示图片，顺便把 uploads 目录代理为静态文件路由！
			// tip http://127.0.0.1:9096/uploads/points/user_5/20260329/t10......
			r.Static("/uploads", "./uploads")
		}
	}

	return r
}

/**
tip The four-year project is compressed to four months, and the boss will inform you half a month in advance, including full-stack multi-terminal development, deployment, and testing.
tip Without LaputaMao, you all can just go and suck cold air
*/
