package service

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
	"errors"
)

// CreateProject 创建项目 (二管调用)
func CreateProject(name, desc string, creatorID uint) (*model.Project, error) {
	project := model.Project{
		Name:        name,
		Description: desc,
		CreatorID:   creatorID,
	}
	if err := config.DB.Create(&project).Error; err != nil {
		return nil, errors.New("创建项目失败: " + err.Error())
	}
	return &project, nil
}

// AssignWorkspace 分配工作区 (二管调用)
//func AssignWorkspace(projectID uint, workspaceName, assigneeUsername, desc string, secAdminID uint) (*model.Workspace, error) {
//	// 1. 明确被分配对象：根据输入的三管 username 查找该用户
//	var thirdAdmin model.User
//	if err := config.DB.Where("username = ? AND role = ?", assigneeUsername, "third_admin").First(&thirdAdmin).Error; err != nil {
//		return nil, errors.New("未找到对应的三级管理员账号")
//	}
//
//	// 安全校验：确认这个三管是不是归当前二管管辖的
//	if thirdAdmin.CreatorID != secAdminID {
//		return nil, errors.New("越权操作：该三级管理员不属于您管辖")
//	}
//
//	// 2. 确认项目是否存在且属于当前二管
//	var project model.Project
//	if err := config.DB.Where("id = ? AND creator_id = ?", projectID, secAdminID).First(&project).Error; err != nil {
//		return nil, errors.New("项目不存在或您无权操作该项目")
//	}
//
//	// 3. 模拟“系统内预置固定的shp范围”：根据 name 拼接系统已存在的路径
//	// tip 部署的时候修改
//	// tip mockFileUrl := fmt.Sprintf("https://your-oss-bucket.com/shps/%s.zip", workspaceName)
//
//	mockFileUrl := fmt.Sprintf("http://127.0.0.1:9096/downloads/%s.zip", workspaceName)
//
//	// 4. 创建工作区并发放
//	workspace := model.Workspace{
//		ProjectID:   project.ID,
//		AssigneeID:  thirdAdmin.ID,
//		Name:        workspaceName,
//		Description: desc,
//		FileUrl:     mockFileUrl,
//		IsRead:      false, // 默认为未读，三管登录后会看到提示
//	}
//
//	if err := config.DB.Create(&workspace).Error; err != nil {
//		return nil, errors.New("分配工作区失败: " + err.Error())
//	}
//
//	return &workspace, nil
//}

// GetMyWorkspaces 获取我的工作区 (三管调用)
// 可用于前端轮询检测新任务，也是列表展示的数据源
func GetMyWorkspaces(thirdAdminID uint) ([]model.Workspace, error) {
	var workspaces []model.Workspace
	// 按时间倒序，最新的任务在最上面
	err := config.DB.Where("assignee_id = ?", thirdAdminID).Order("created_at desc").Find(&workspaces).Error
	return workspaces, err
}

// MarkWorkspaceAsRead 标记工作区为已读 (三管点开任务详情时自动调用)
func MarkWorkspaceAsRead(workspaceID, thirdAdminID uint) error {
	res := config.DB.Model(&model.Workspace{}).
		Where("id = ? AND assignee_id = ?", workspaceID, thirdAdminID).
		Update("is_read", true)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("工作区任务不存在或无权操作")
	}
	return nil
}
