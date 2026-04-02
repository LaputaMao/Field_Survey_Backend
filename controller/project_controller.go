package controller

import (
	"net/http"
	"strconv"

	"Field_Survey_Backend/service"

	"github.com/gin-gonic/gin"
)

// ---- [二管相关接口] ----

type CreateProjectReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func CreateProjectHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	var req CreateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，缺失项目名称"})
		return
	}

	project, err := service.CreateProject(req.Name, req.Description, creatorID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "项目创建成功", "data": project})
}

//type AssignWorkspaceReq struct {
//	ProjectID          uint   `json:"project_id" binding:"required"`
//	WorkspaceName      string `json:"workspace_name" binding:"required"`
//	ThirdAdminUsername string `json:"third_admin_username" binding:"required"`
//	Description        string `json:"description"`
//}
//
//func AssignWorkspaceHandler(c *gin.Context) {
//	secAdminID, _ := c.Get("userID")
//	var req AssignWorkspaceReq
//	if err := c.ShouldBindJSON(&req); err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误，请检查输入"})
//		return
//	}
//
//	workspace, err := service.AssignWorkspace(req.ProjectID, req.WorkspaceName, req.ThirdAdminUsername, req.Description, secAdminID.(uint))
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
//		return
//	}
//	c.JSON(http.StatusOK, gin.H{"message": "任务工作区下发成功", "data": workspace})
//}

// ---- [三管相关接口] ----

func GetMyWorkspacesHandler(c *gin.Context) {
	thirdAdminID, _ := c.Get("userID")
	workspaces, err := service.GetMyWorkspaces(thirdAdminID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取工作区任务失败"})
		return
	}

	// 顺便统计一下有多少个未读任务，方便前端右上角直接显示小红点数量
	unreadCount := 0
	for _, w := range workspaces {
		if !w.IsRead {
			unreadCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":         workspaces,
		"unread_count": unreadCount,
	})
}

func ReadWorkspaceHandler(c *gin.Context) {
	thirdAdminID, _ := c.Get("userID")
	workspaceIDStr := c.Param("id")
	workspaceID, err := strconv.Atoi(workspaceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	err = service.MarkWorkspaceAsRead(uint(workspaceID), thirdAdminID.(uint))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "状态标为已读"})
}
