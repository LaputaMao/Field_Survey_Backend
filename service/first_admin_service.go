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

//// ImportUsersHierarchyFromExcel 导入人员表
//// Excel格式：姓名, 电话, 二级项目名, 工作区名, 职责
//// 职责枚举：一级, 二级, 三级, 调查员
//func ImportUsersHierarchyFromExcel(file multipart.File, creatorID uint) (*ImportUsersStats, error) {
//	stats := &ImportUsersStats{
//		Details: []string{},
//	}
//
//	// 读取 Excel 文件
//	f, err := excelize.OpenReader(file)
//	if err != nil {
//		return nil, errors.New("无法读取Excel文件: " + err.Error())
//	}
//	defer f.Close()
//
//	// 假设数据在第一张表 "Sheet1"
//	rows, err := f.GetRows("Sheet1")
//	if err != nil {
//		return nil, errors.New("无法读取Sheet1数据: " + err.Error())
//	}
//
//	// 开启事务
//	tx := config.DB.Begin()
//	defer func() {
//		if r := recover(); r != nil {
//			tx.Rollback()
//		}
//	}()
//
//	// 预加载项目和工作区映射
//	var projects []model.Project
//	var workspaces []model.Workspace
//	if err := tx.Find(&projects).Error; err != nil {
//		tx.Rollback()
//		return nil, fmt.Errorf("加载项目列表失败: %v", err)
//	}
//	if err := tx.Find(&workspaces).Error; err != nil {
//		tx.Rollback()
//		return nil, fmt.Errorf("加载工作区列表失败: %v", err)
//	}
//
//	projectMap := make(map[string]uint) // 项目名 -> 项目ID
//	for _, p := range projects {
//		projectMap[p.Name] = p.ID
//	}
//
//	workspaceMap := make(map[string]uint) // 组合键 "项目ID_工作区名" -> 工作区ID
//	for _, w := range workspaces {
//		key := fmt.Sprintf("%d_%s", w.ProjectID, w.Name)
//		workspaceMap[key] = w.ID
//	}
//
//	// 密码生成函数（使用电话作为密码）
//	generatePassword := func(phone string) string {
//		return phone
//	}
//
//	for i, row := range rows {
//		lineNum := i + 1
//		if i == 0 {
//			continue // 跳过表头
//		}
//		if len(row) < 5 {
//			stats.Details = append(stats.Details, fmt.Sprintf("第%d行数据不足5列，已跳过", lineNum))
//			continue
//		}
//
//		username := strings.TrimSpace(row[0])
//		phone := strings.TrimSpace(row[1])
//		projectName := strings.TrimSpace(row[2])
//		workspaceName := strings.TrimSpace(row[3])
//		roleStr := strings.TrimSpace(row[4])
//
//		if username == "" {
//			stats.Details = append(stats.Details, fmt.Sprintf("第%d行用户名为空，已跳过", lineNum))
//			continue
//		}
//		if phone == "" {
//			stats.Details = append(stats.Details, fmt.Sprintf("第%d行电话为空，已跳过", lineNum))
//			continue
//		}
//
//		// 映射角色
//		var role string
//		switch roleStr {
//		case "一级":
//			role = "first_admin"
//		case "二级":
//			role = "sec_admin"
//		case "三级":
//			role = "third_admin"
//		case "调查员":
//			role = "user"
//		default:
//			stats.Details = append(stats.Details, fmt.Sprintf("第%d行角色 '%s' 无效，应为：一级、二级、三级、调查员，已跳过", lineNum, roleStr))
//			continue
//		}
//
//		// 检查用户名是否已存在
//		var existingUser model.User
//		err := tx.Where("username = ?", username).First(&existingUser).Error
//		if err == nil {
//			stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 已存在，跳过", username))
//			continue
//		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
//			tx.Rollback()
//			return nil, fmt.Errorf("查询用户 '%s' 失败: %v", username, err)
//		}
//
//		// 创建用户
//		plainPassword := generatePassword(phone)
//		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
//		if err != nil {
//			tx.Rollback()
//			return nil, fmt.Errorf("加密密码失败: %v", err)
//		}
//
//		user := model.User{
//			Username:  username,
//			Email:     "", // 可留空
//			Phone:     phone,
//			Password:  string(hashedPassword),
//			Role:      role,
//			CreatorID: creatorID, // 一级管理员作为创建者
//		}
//
//		if err := tx.Create(&user).Error; err != nil {
//			tx.Rollback()
//			return nil, fmt.Errorf("创建用户 '%s' 失败: %v", username, err)
//		}
//		stats.UsersCreated++
//		stats.Details = append(stats.Details, fmt.Sprintf("创建用户 '%s'，角色: %s，初始密码: %s", username, roleStr, plainPassword))
//
//		// 根据角色关联项目和工作区
//		if role == "sec_admin" {
//			// 二级管理员关联项目（作为项目创建者）
//			if projectName == "" {
//				stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 是二级管理员但未指定项目，跳过关联", username))
//				continue
//			}
//			projectID, exists := projectMap[projectName]
//			if !exists {
//				stats.Details = append(stats.Details, fmt.Sprintf("项目 '%s' 不存在，用户 '%s' 关联失败", projectName, username))
//				continue
//			}
//			// 更新项目的CreatorID为该二级管理员
//			if err := tx.Model(&model.Project{}).Where("id = ?", projectID).Update("creator_id", user.ID).Error; err != nil {
//				tx.Rollback()
//				return nil, fmt.Errorf("更新项目创建者失败: %v", err)
//			}
//			stats.ProjectsLinked++
//			stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 关联项目 '%s'", username, projectName))
//		} else if role == "third_admin" {
//			// 三级管理员关联工作区（作为工作区负责人）
//			if projectName == "" || workspaceName == "" {
//				stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 是三级管理员但未指定项目/工作区，跳过关联", username))
//				continue
//			}
//			projectID, exists := projectMap[projectName]
//			if !exists {
//				stats.Details = append(stats.Details, fmt.Sprintf("项目 '%s' 不存在，用户 '%s' 关联失败", projectName, username))
//				continue
//			}
//			key := fmt.Sprintf("%d_%s", projectID, workspaceName)
//			workspaceID, exists := workspaceMap[key]
//			if !exists {
//				stats.Details = append(stats.Details, fmt.Sprintf("工作区 '%s' 在项目 '%s' 中不存在，用户 '%s' 关联失败", workspaceName, projectName, username))
//				continue
//			}
//			// 更新工作区的AssigneeID为该三级管理员
//			if err := tx.Model(&model.Workspace{}).Where("id = ?", workspaceID).Update("assignee_id", user.ID).Error; err != nil {
//				tx.Rollback()
//				return nil, fmt.Errorf("更新工作区负责人失败: %v", err)
//			}
//			stats.WorkspacesLinked++
//			stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 关联工作区 '%s'（项目: %s）", username, workspaceName, projectName))
//		} else if role == "user" {
//			// 调查员不需要在此处关联，他们通过任务分配关联
//			// 但可以记录其所属项目和工作区信息（可存储在用户扩展字段中，这里暂不处理）
//			stats.Details = append(stats.Details, fmt.Sprintf("调查员 '%s' 创建成功，需后续分配任务", username))
//		}
//		// first_admin 角色不需要特殊处理
//	}
//
//	if err := tx.Commit().Error; err != nil {
//		return nil, fmt.Errorf("提交事务失败: %v", err)
//	}
//
//	return stats, nil
//}

// ImportUsersHierarchyFromExcel 导入人员表
// Excel格式：姓名, 电话, 二级项目名, 工作区名, 职责
// 职责枚举：一级, 二级, 三级, 调查员
func ImportUsersHierarchyFromExcel(file multipart.File, creatorID uint) (*ImportUsersStats, error) {
	stats := &ImportUsersStats{
		Details: []string{},
	}

	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, errors.New("无法读取Excel文件: " + err.Error())
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, errors.New("无法读取Sheet1数据: " + err.Error())
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

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

	projectMap := make(map[string]uint)
	for _, p := range projects {
		projectMap[p.Name] = p.ID
	}

	workspaceMap := make(map[string]uint)
	for _, w := range workspaces {
		key := fmt.Sprintf("%d_%s", w.ProjectID, w.Name)
		workspaceMap[key] = w.ID
	}

	// ⭐ 新增：定义一个结构来暂存本次添加的调查员，方便最后统一绑定
	type PendingSurveyor struct {
		UserID        uint
		Username      string
		ProjectName   string
		WorkspaceName string
	}
	var pendingSurveyors []PendingSurveyor

	generatePassword := func(phone string) string {
		return phone
	}

	for i, row := range rows {
		lineNum := i + 1
		if i == 0 {
			continue
		}
		if len(row) < 5 {
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行数据不足5列，已跳过", lineNum))
			continue
		}

		username := strings.TrimSpace(row[0])
		phone := strings.TrimSpace(row[1])
		projectName := strings.TrimSpace(row[2])
		workspaceName := strings.TrimSpace(row[3])
		roleStr := strings.TrimSpace(row[4])

		if username == "" {
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行用户名为空，已跳过", lineNum))
			continue
		}
		if phone == "" {
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行电话为空，已跳过", lineNum))
			continue
		}

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
			stats.Details = append(stats.Details, fmt.Sprintf("第%d行角色 '%s' 无效，跳过", lineNum, roleStr))
			continue
		}

		var existingUser model.User
		err := tx.Where("username = ?", username).First(&existingUser).Error
		if err == nil {
			stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 已存在，跳过建号", username))
			// 可选：如果你希望已存在的调查员也能随着重新导表更新归属，也可以把 existingUser 加入 pendingSurveyors。
			// 这里我们遵循你原先逻辑：已存在用户整体跳过。
			continue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return nil, fmt.Errorf("查询用户 '%s' 失败: %v", username, err)
		}

		plainPassword := generatePassword(phone)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("加密密码失败: %v", err)
		}

		user := model.User{
			Username:  username,
			Phone:     phone,
			Password:  string(hashedPassword),
			Role:      role,
			CreatorID: creatorID,
		}

		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("创建用户 '%s' 失败: %v", username, err)
		}
		stats.UsersCreated++
		stats.Details = append(stats.Details, fmt.Sprintf("创建用户 '%s'，角色: %s", username, roleStr))

		// 处理不同角色的项目与工作区关联
		if role == "sec_admin" {
			if projectName == "" {
				stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 是二级但未指定项目，跳过关联", username))
				continue
			}
			projectID, exists := projectMap[projectName]
			if !exists {
				stats.Details = append(stats.Details, fmt.Sprintf("项目 '%s' 不存在，用户 '%s' 关联失败", projectName, username))
				continue
			}
			if err := tx.Model(&model.Project{}).Where("id = ?", projectID).Update("creator_id", user.ID).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("更新项目创建者失败: %v", err)
			}
			stats.ProjectsLinked++
		} else if role == "third_admin" {
			if projectName == "" || workspaceName == "" {
				stats.Details = append(stats.Details, fmt.Sprintf("用户 '%s' 是三级但未指定项目/工作区，跳过关联", username))
				continue
			}
			projectID, exists := projectMap[projectName]
			if !exists {
				stats.Details = append(stats.Details, fmt.Sprintf("项目 '%s' 不存在", projectName))
				continue
			}
			key := fmt.Sprintf("%d_%s", projectID, workspaceName)
			workspaceID, exists := workspaceMap[key]
			if !exists {
				stats.Details = append(stats.Details, fmt.Sprintf("工作区 '%s' 不存在", workspaceName))
				continue
			}
			if err := tx.Model(&model.Workspace{}).Where("id = ?", workspaceID).Update("assignee_id", user.ID).Error; err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("更新工作区负责人失败: %v", err)
			}
			stats.WorkspacesLinked++
		} else if role == "user" {
			// ⭐ 新增记录逻辑：将刚创建的调查员和表格填写的名录记录下来，放进待处理列表
			pendingSurveyors = append(pendingSurveyors, PendingSurveyor{
				UserID:        user.ID,
				Username:      user.Username,
				ProjectName:   projectName,
				WorkspaceName: workspaceName,
			})
		}
	}

	// =========================================================================
	// ⭐ 新增阶段：循环表外处理（调查员寻找三管老板认亲环节）
	// 为何放这里？因为如果 Excel 表里三管在调查员行下面才被创建，等到循环跑完这里时，工作区表肯定已经绑定好三管的 ID 了！
	// =========================================================================
	for _, ps := range pendingSurveyors {
		if ps.ProjectName == "" || ps.WorkspaceName == "" {
			stats.Details = append(stats.Details, fmt.Sprintf("调查员 '%s' 未填写关联项目与工作区，无法分配直属三管，将被置入暂不可派发状态", ps.Username))
			continue
		}

		projectID, exists := projectMap[ps.ProjectName]
		if !exists {
			stats.Details = append(stats.Details, fmt.Sprintf("调查员 '%s' 填写的项目 '%s' 未找到", ps.Username, ps.ProjectName))
			continue
		}

		key := fmt.Sprintf("%d_%s", projectID, ps.WorkspaceName)
		workspaceID, exists := workspaceMap[key]
		if !exists {
			stats.Details = append(stats.Details, fmt.Sprintf("调查员 '%s' 填写的工作区 '%s' 未找到", ps.Username, ps.WorkspaceName))
			continue
		}

		// 查出这个工作区目前的三管是谁
		var ws model.Workspace
		if err := tx.Where("id = ?", workspaceID).First(&ws).Error; err != nil {
			continue // 处理极特殊情况
		}

		if ws.AssigneeID == 0 {
			stats.Details = append(stats.Details, fmt.Sprintf("调查员 '%s' 的工作区 '%s' 暂未配置三管负责人，绑定遗落", ps.Username, ps.WorkspaceName))
			continue
		}

		// 核心：把三管的老板 ID 赋予给调查员的 CreatorID
		if err := tx.Model(&model.User{}).Where("id = ?", ps.UserID).Update("creator_id", ws.AssigneeID).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("调查员归属更新失败: %v", err)
		}

		stats.Details = append(stats.Details, fmt.Sprintf("调查员 '%s' 成功编入工作区 '%s'，直属三管系统已确立", ps.Username, ps.WorkspaceName))
	}

	// 最终提交全部！
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %v", err)
	}

	return stats, nil
}
