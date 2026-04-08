package controller

import (
	"Field_Survey_Backend/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ImportProjectsHandler 导入项目表（二级项目和工作区）
func ImportProjectsHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")

	// 只有一级管理员可以导入
	if creatorRole.(string) != "first_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅一级管理员可导入项目"})
		return
	}

	fileHeader, err := c.FormFile("projects_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传项目Excel文件 (参数名: projects_file)"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件打开失败"})
		return
	}
	defer file.Close()

	stats, err := service.ImportProjectsFromExcel(file, creatorID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "项目导入成功",
		"projects_created":   stats.ProjectsCreated,
		"workspaces_created": stats.WorkspacesCreated,
		"details":            stats.Details,
	})
}

// ImportUsersHandler 导入人员表
func ImportUsersHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")

	if creatorRole.(string) != "first_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅一级管理员可导入人员"})
		return
	}

	fileHeader, err := c.FormFile("users_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传人员Excel文件 (参数名: users_file)"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件打开失败"})
		return
	}
	defer file.Close()

	stats, err := service.ImportUsersHierarchyFromExcel(file, creatorID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "人员导入成功",
		"users_created":     stats.UsersCreated,
		"projects_linked":   stats.ProjectsLinked,
		"workspaces_linked": stats.WorkspacesLinked,
		"details":           stats.Details,
	})
}

// CombinedImportHandler 合并导入项目和人员（两个文件同时上传）
func CombinedImportHandler(c *gin.Context) {
	creatorID, _ := c.Get("userID")
	creatorRole, _ := c.Get("role")

	if creatorRole.(string) != "first_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，仅一级管理员可导入"})
		return
	}

	projectsFileHeader, err := c.FormFile("projects_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传项目Excel文件 (参数名: projects_file)"})
		return
	}
	usersFileHeader, err := c.FormFile("users_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传人员Excel文件 (参数名: users_file)"})
		return
	}

	projectsFile, err := projectsFileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "项目文件打开失败"})
		return
	}
	defer projectsFile.Close()

	usersFile, err := usersFileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "人员文件打开失败"})
		return
	}
	defer usersFile.Close()

	// 先导入项目，再导入人员
	projectStats, err := service.ImportProjectsFromExcel(projectsFile, creatorID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "项目导入失败: " + err.Error()})
		return
	}

	userStats, err := service.ImportUsersHierarchyFromExcel(usersFile, creatorID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "人员导入失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "合并导入成功",
		"projects": gin.H{
			"created":            projectStats.ProjectsCreated,
			"workspaces_created": projectStats.WorkspacesCreated,
			"details":            projectStats.Details,
		},
		"users": gin.H{
			"created":           userStats.UsersCreated,
			"projects_linked":   userStats.ProjectsLinked,
			"workspaces_linked": userStats.WorkspacesLinked,
			"details":           userStats.Details,
		},
	})
}
