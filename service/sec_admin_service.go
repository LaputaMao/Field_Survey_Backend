package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
	"Field_Survey_Backend/utils"
)

// UploadBasicShp ========= Req 1: 上传全国基础生态区 SHP =========
func UploadBasicShp(file multipart.File, fileHeader *multipart.FileHeader) error {
	tempZipPath := fmt.Sprintf("./uploads/temp_basic_%d.zip", time.Now().Unix())
	tempZipFile, err := os.Create(tempZipPath)
	if err != nil {
		return errors.New("服务器临时文件创建失败")
	}
	io.Copy(tempZipFile, file)
	tempZipFile.Close()
	defer os.Remove(tempZipPath)

	// 解压到 ./uploads/basic/{文件名(去后缀)}
	folderName := strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
	destDir := fmt.Sprintf("./uploads/basic/%s", folderName)

	// Utils 之前写的 UnzipAndFindShp 会找出一个主 .shp 路径
	_, err = utils.UnzipAndFindShp(tempZipPath, destDir)
	if err != nil {
		return fmt.Errorf("解压解析基础底图失败: %v", err)
	}
	return nil
}

// GetBasicShpList ========= Req 2: 获取所有的 Basic SHP 列表 =========
func GetBasicShpList() ([]string, error) {
	var shpPaths []string
	baseDir := "./uploads/basic"

	// 如果目录不存在说明还没上传过，返回空数组
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return shpPaths, nil
	}

	// 遍历 basic 目录，寻找所有的 .shp 文件
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".shp") {
			// 将 "\" 统一转为 "/" 方便前端展示
			shpPaths = append(shpPaths, filepath.ToSlash(path))
		}
		return nil
	})

	return shpPaths, err
}

// AssignWorkspace ========= Req 4 (重构): 裁切底图并派发工作区 =========
func AssignWorkspace(projectID uint, workspaceName, assigneeUsername, desc, basicShpPath string, secAdminID uint) (*model.Workspace, error) {
	var thirdAdmin model.User
	if err := config.DB.Where("username = ? AND role = ?", assigneeUsername, "third_admin").First(&thirdAdmin).Error; err != nil {
		return nil, errors.New("未找到三级管理员")
	}
	if thirdAdmin.CreatorID != secAdminID {
		return nil, errors.New("越权操作")
	}

	var project model.Project
	if err := config.DB.Where("id = ? AND creator_id = ?", projectID, secAdminID).First(&project).Error; err != nil {
		return nil, errors.New("项目不存在或越权")
	}

	// ⭐ 核心逻辑：从全国底图抠出这个工作区，存到独立的三管专属文件夹
	// 格式：./uploads/workspaces/{三管名}_{工作区名}_{时间戳}
	outDir := fmt.Sprintf("./uploads/workspaces/%s_%s_%d", thirdAdmin.Username, workspaceName, time.Now().Unix())

	// 调用工具进行要素提取 (注意：属性字段名 "三级名" 必须与底图完全一致)
	extractedShpPath, err := utils.ExtractFeatureToNewShp(basicShpPath, outDir, "三级名", workspaceName)
	if err != nil {
		return nil, fmt.Errorf("地理空间分配失败: %v", err)
	}

	workspace := model.Workspace{
		ProjectID:   project.ID,
		AssigneeID:  thirdAdmin.ID,
		Name:        workspaceName,
		Description: desc,
		// 将本地路径格式化为前端可直接访问的静态路由格式
		FileUrl: "/" + filepath.ToSlash(extractedShpPath),
		IsRead:  false,
	}

	if err := config.DB.Create(&workspace).Error; err != nil {
		return nil, errors.New("落库失败")
	}

	return &workspace, nil
}
