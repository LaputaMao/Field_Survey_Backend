package service

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ImportProjectsStats 项目导入统计
type ImportProjectsStats struct {
	ProjectsCreated   int      `json:"projects_created"`
	WorkspacesCreated int      `json:"workspaces_created"`
	Details           []string `json:"details"`
}

// ImportUsersStats 人员导入统计
type ImportUsersStats struct {
	UsersCreated     int      `json:"users_created"`
	ProjectsLinked   int      `json:"projects_linked"`
	WorkspacesLinked int      `json:"workspaces_linked"`
	Details          []string `json:"details"`
}

// ImportProjectsFromExcel 导入项目表
// Excel格式：第一列二级项目名，第二列工作区名
func ImportProjectsFromExcel(file multipart.File, creatorID uint) (*ImportProjectsStats, error) {
	stats := &ImportProjectsStats{
		Details: []string{},
	}

	// 读取 Excel 文件
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, errors.New("无法读取Excel文件: " + err.Error())
	}
	defer f.Close()

	// 假设数据在第一张表 "Sheet1"
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, errors.New("无法读取Sheet1数据: " + err.Error())
	}

	// 用于去重记录
	projectMap := make(map[string]uint)   // 项目名 -> 项目ID
	workspaceMap := make(map[string]bool) // 工作区名 -> 是否存在（工作区名可能全局唯一？暂定项目内唯一）

	// 开启事务
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for i, row := range rows {
		lineNum := i + 1
		if i == 0 {
			continue // 跳过表头
		}
		if len(row) < 2 {
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行数据不足2列，已跳过", lineNum))
			continue
		}

		projectName := strings.TrimSpace(row[0])
		workspaceName := strings.TrimSpace(row[1])

		if projectName == "" {
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行项目名为空，已跳过", lineNum))
			continue
		}
		if workspaceName == "" {
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行工作区名为空，已跳过", lineNum))
			continue
		}

		// 检查或创建项目
		var projectID uint
		if pid, exists := projectMap[projectName]; exists {
			projectID = pid
		} else {
			// 查询是否已存在同名项目
			var existingProject model.Project
			err := tx.Where("name = ?", projectName).First(&existingProject).Error
			if err == nil {
				// 项目已存在
				projectID = existingProject.ID
				projectMap[projectName] = projectID
				stats.Details = append(stats.Details, fmt.Sprintf("项目 '%s' 已存在，使用现有ID: %d", projectName, projectID))
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// 创建新项目
				project := model.Project{
					Name:        projectName,
					Description: "由一级管理员导入",
					CreatorID:   creatorID, // 一级管理员作为创建者
				}
				if err := tx.Create(&project).Error; err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("创建项目 '%s' 失败: %v", projectName, err)
				}
				projectID = project.ID
				projectMap[projectName] = projectID
				stats.ProjectsCreated++
				stats.Details = append(stats.Details, fmt.Sprintf("创建项目 '%s'，ID: %d", projectName, projectID))
			} else {
				tx.Rollback()
				return nil, fmt.Errorf("查询项目 '%s' 失败: %v", projectName, err)
			}
		}

		// 检查工作区是否已存在（在同一个项目内）
		workspaceKey := fmt.Sprintf("%d_%s", projectID, workspaceName)
		if workspaceMap[workspaceKey] {
			stats.Details = append(stats.Details, fmt.Sprintf("项目 '%s' 下的工作区 '%s' 已存在，跳过重复", projectName, workspaceName))
			continue
		}

		// 检查工作区是否已存在（数据库查询）
		var existingWorkspace model.Workspace
		err := tx.Where("project_id = ? AND name = ?", projectID, workspaceName).First(&existingWorkspace).Error
		if err == nil {
			// 工作区已存在
			workspaceMap[workspaceKey] = true
			stats.Details = append(stats.Details, fmt.Sprintf("项目 '%s' 下的工作区 '%s' 已存在，使用现有ID: %d", projectName, workspaceName, existingWorkspace.ID))
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return nil, fmt.Errorf("查询工作区 '%s' 失败: %v", workspaceName, err)
		}

		// 创建新工作区（AssigneeID 暂为空，等导入三级管理员时再分配）
		workspace := model.Workspace{
			ProjectID:   projectID,
			AssigneeID:  0, // 暂未分配
			Name:        workspaceName,
			Description: "由一级管理员导入",
			FileUrl:     "", // 暂无文件
			IsRead:      false,
		}
		if err := tx.Create(&workspace).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建工作区 '%s' 失败: %v", workspaceName, err)
		}
		workspaceMap[workspaceKey] = true
		stats.WorkspacesCreated++
		stats.Details = append(stats.Details, fmt.Sprintf("创建项目 '%s' 下的工作区 '%s'，ID: %d", projectName, workspaceName, workspace.ID))
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %v", err)
	}

	return stats, nil
}

// ImportUsersHierarchyFromExcel 导入人员表
// Excel格式：姓名, 二级项目名, 工作区名, 职责
// 职责枚举：一级, 二级, 三级, 调查员
func ImportUsersHierarchyFromExcel(file multipart.File, creatorID uint) (*ImportUsersStats, error) {
	stats := &ImportUsersStats{
		Details: []string{},
	}

	// 读取 Excel 文件
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, errors.New("无法读取Excel文件: " + err.Error())
	}
	defer f.Close()

	// 假设数据在第一张表 "Sheet1"
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, errors.New("无法读取Sheet1数据: " + err.Error())
	}

	// 开启事务
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 预加载项目和工作区映射
	var projects []model.Project
	var workspaces []model.Workspace
	if err := tx.Find(&projects).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("加载项目列表失败: %v", err)
	}
	if err := tx.Find(&workspaces).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("加载工作区列表失败: %v", err)
	}

	projectMap := make(map[string]uint) // 项目名 -> 项目ID
	for _, p := range projects {
		projectMap[p.Name] = p.ID
	}

	workspaceMap := make(map[string]uint) // 组合键 "项目ID_工作区名" -> 工作区ID
	for _, w := range workspaces {
		key := fmt.Sprintf("%d_%s", w.ProjectID, w.Name)
		workspaceMap[key] = w.ID
	}

	// 密码生成函数（默认密码为姓名+123）
	generatePassword := func(username string) string {
		return username + "123"
	}

	for i, row := range rows {
		lineNum := i + 1
		if i == 0 {
			continue // 跳过表头
		}
		if len(row) < 4 {
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行数据不足4列，已跳过", lineNum))
			continue
		}

		username := strings.TrimSpace(row[0])
		projectName := strings.TrimSpace(row[1])
		workspaceName := strings.TrimSpace(row[2])
		roleStr := strings.TrimSpace(row[3])

		if username == "" {
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行用户名为空，已跳过", lineNum))
			continue
		}

		// 映射角色
		var role string
		switch roleStr {
		case "一级":
			role = "first_admin"
		case "二级":
			role = "sec_admin"
		case "三级":
			role = "third_admin"
		case "调查员":
			role = "user"
		default:
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行角色 '%s' 无效，应为：一级、二级、三级、调查员，已跳过", lineNum, roleStr))
			continue
		}

		// 检查用户名是否已存在
		var existingUser model.User
		err := tx.Where("username = ?", username).First(&existingUser).Error
		if err == nil {
			stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 已存在，跳过", username))
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return nil, fmt.Errorf("查询用户 '%s' 失败: %v", username, err)
		}

		// 创建用户
		plainPassword := generatePassword(username)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("加密密码失败: %v", err)
		}

		user := model.User{
			Username:  username,
			Email:     "", // 可留空
			Password:  string(hashedPassword),
			Role:      role,
			CreatorID: creatorID, // 一级管理员作为创建者
		}

		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建用户 '%s' 失败: %v", username, err)
		}
		stats.UsersCreated++
		stats.Details = append(stats.Details, fmt.Sprintf("创建用户 '%s'，角色: %s，初始密码: %s", username, roleStr, plainPassword))

		// 根据角色关联项目和工作区
		if role == "sec_admin" {
			// 二级管理员关联项目（作为项目创建者）
			if projectName == "" {
				stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 是二级管理员但未指定项目，跳过关联", username))
				continue
			}
			projectID, exists := projectMap[projectName]
			if !exists {
				stats.Details = append(stats.Details, fmt.Sprintf("项目 '%s' 不存在，用户 '%s' 关联失败", projectName, username))
				continue
			}
			// 更新项目的CreatorID为该二级管理员
			if err := tx.Model(&model.Project{}).Where("id = ?", projectID).Update("creator_id", user.ID).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("更新项目创建者失败: %v", err)
			}
			stats.ProjectsLinked++
			stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 关联项目 '%s'", username, projectName))
		} else if role == "third_admin" {
			// 三级管理员关联工作区（作为工作区负责人）
			if projectName == "" || workspaceName == "" {
				stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 是三级管理员但未指定项目/工作区，跳过关联", username))
				continue
			}
			projectID, exists := projectMap[projectName]
			if !exists {
				stats.Details = append(stats.Details, fmt.Sprintf("项目 '%s' 不存在，用户 '%s' 关联失败", projectName, username))
				continue
			}
			key := fmt.Sprintf("%d_%s", projectID, workspaceName)
			workspaceID, exists := workspaceMap[key]
			if !exists {
				stats.Details = append(stats.Details, fmt.Sprintf("工作区 '%s' 在项目 '%s' 中不存在，用户 '%s' 关联失败", workspaceName, projectName, username))
				continue
			}
			// 更新工作区的AssigneeID为该三级管理员
			if err := tx.Model(&model.Workspace{}).Where("id = ?", workspaceID).Update("assignee_id", user.ID).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("更新工作区负责人失败: %v", err)
			}
			stats.WorkspacesLinked++
			stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 关联工作区 '%s'（项目: %s）", username, workspaceName, projectName))
		} else if role == "user" {
			// 调查员不需要在此处关联，他们通过任务分配关联
			// 但可以记录其所属项目和工作区信息（可存储在用户扩展字段中，这里暂不处理）
			stats.Details = append(stats.Details, fmt.Sprintf("调查员 '%s' 创建成功，需后续分配任务", username))
		}
		// first_admin 角色不需要特殊处理
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %v", err)
	}

	return stats, nil
}
