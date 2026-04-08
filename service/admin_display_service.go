package service

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/dto"
	"Field_Survey_Backend/model"
	"Field_Survey_Backend/utils"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ================ 项目工作区树状结构 ================

// ProjectTreeItem 项目树节点
type ProjectTreeItem struct {
	ProjectID   uint                `json:"project_id"`
	ProjectName string              `json:"project_name"`
	Workspaces  []WorkspaceTreeItem `json:"workspaces"`
}

// WorkspaceTreeItem 工作区树节点
type WorkspaceTreeItem struct {
	WorkspaceID   uint               `json:"workspace_id"`
	WorkspaceName string             `json:"workspace_name"`
	AssigneeID    uint               `json:"assignee_id,omitempty"`
	AssigneeName  string             `json:"assignee_name,omitempty"`
	Surveyors     []dto.SurveyorItem `json:"surveyors,omitempty"` // 工作区下的调查员列表
}

// GetProjectsTreeForFirstAdmin 一级管理员获取所有项目和工作区树
func GetProjectsTreeForFirstAdmin() ([]ProjectTreeItem, error) {
	var projects []model.Project
	if err := config.DB.Find(&projects).Error; err != nil {
		return nil, errors.New("查询项目失败: " + err.Error())
	}

	return buildProjectsTree(projects, nil, nil)
}

// GetProjectsTreeForSecAdmin 二级管理员获取管辖的项目和工作区树
func GetProjectsTreeForSecAdmin(secAdminID uint) ([]ProjectTreeItem, error) {
	// 查询该二级管理员创建的项目
	var projects []model.Project
	if err := config.DB.Where("creator_id = ?", secAdminID).Find(&projects).Error; err != nil {
		return nil, errors.New("查询项目失败: " + err.Error())
	}

	if len(projects) == 0 {
		return []ProjectTreeItem{}, nil
	}

	// 获取项目ID列表
	var projectIDs []uint
	for _, p := range projects {
		projectIDs = append(projectIDs, p.ID)
	}

	return buildProjectsTree(projects, projectIDs, nil)
}

// GetWorkspacesTreeForThirdAdmin 三级管理员获取负责的工作区树（兼容现有接口）
func GetWorkspacesTreeForThirdAdmin(thirdAdminID uint) ([]dto.WorkspaceListResponse, error) {
	var workspaces []model.Workspace
	if err := config.DB.Where("assignee_id = ?", thirdAdminID).Find(&workspaces).Error; err != nil {
		return nil, errors.New("查询工作区失败")
	}

	var results []dto.WorkspaceListResponse
	for _, ws := range workspaces {
		wsItem := dto.WorkspaceListResponse{
			WorkspaceID:   ws.ID,
			WorkspaceName: ws.Name,
			Surveyors:     []dto.SurveyorItem{},
		}

		// 获取该工作区下的调查员（通过任务分配）
		type Result struct {
			TaskID   uint
			UserID   uint
			Username string
		}
		var rows []Result
		config.DB.Table("tasks t").
			Select("t.id as task_id, u.id as user_id, u.username").
			Joins("JOIN users u ON t.assignee_id = u.id").
			Where("t.workspace_id = ?", ws.ID).
			Scan(&rows)

		for _, r := range rows {
			wsItem.Surveyors = append(wsItem.Surveyors, dto.SurveyorItem{
				TaskID:   r.TaskID,
				UserID:   r.UserID,
				Username: r.Username,
			})
		}
		results = append(results, wsItem)
	}

	return results, nil
}

// buildProjectsTree 构建项目工作区树通用方法
func buildProjectsTree(projects []model.Project, filterProjectIDs []uint, filterWorkspaceAssigneeID *uint) ([]ProjectTreeItem, error) {
	var projectTree []ProjectTreeItem

	// 如果没有项目，返回空
	if len(projects) == 0 {
		return projectTree, nil
	}

	// 构建项目ID映射
	projectMap := make(map[uint]*ProjectTreeItem)
	for _, p := range projects {
		projectMap[p.ID] = &ProjectTreeItem{
			ProjectID:   p.ID,
			ProjectName: p.Name,
			Workspaces:  []WorkspaceTreeItem{},
		}
	}

	// 查询工作区
	var workspaces []model.Workspace
	query := config.DB
	if len(filterProjectIDs) > 0 {
		query = query.Where("project_id IN (?)", filterProjectIDs)
	}
	if filterWorkspaceAssigneeID != nil {
		query = query.Where("assignee_id = ?", *filterWorkspaceAssigneeID)
	}
	if err := query.Find(&workspaces).Error; err != nil {
		return nil, errors.New("查询工作区失败: " + err.Error())
	}

	// 获取工作区负责人信息
	workspaceIDs := make([]uint, 0, len(workspaces))
	for _, ws := range workspaces {
		workspaceIDs = append(workspaceIDs, ws.ID)
	}

	// 查询工作区负责人信息
	var assignees []struct {
		WorkspaceID uint
		UserID      uint
		Username    string
	}
	if len(workspaceIDs) > 0 {
		config.DB.Table("workspaces w").
			Select("w.id as workspace_id, u.id as user_id, u.username").
			Joins("LEFT JOIN users u ON w.assignee_id = u.id").
			Where("w.id IN (?)", workspaceIDs).
			Scan(&assignees)
	}

	assigneeMap := make(map[uint]struct {
		UserID   uint
		Username string
	})
	for _, a := range assignees {
		assigneeMap[a.WorkspaceID] = struct {
			UserID   uint
			Username string
		}{UserID: a.UserID, Username: a.Username}
	}

	// 查询每个工作区下的调查员（通过任务）
	workspaceSurveyorMap := make(map[uint][]dto.SurveyorItem)
	if len(workspaceIDs) > 0 {
		type SurveyorResult struct {
			WorkspaceID uint
			TaskID      uint
			UserID      uint
			Username    string
		}
		var surveyorRows []SurveyorResult
		config.DB.Table("tasks t").
			Select("t.workspace_id as workspace_id, t.id as task_id, u.id as user_id, u.username").
			Joins("JOIN users u ON t.assignee_id = u.id").
			Where("t.workspace_id IN (?)", workspaceIDs).
			Scan(&surveyorRows)

		for _, row := range surveyorRows {
			wsItem := dto.SurveyorItem{
				TaskID:   row.TaskID,
				UserID:   row.UserID,
				Username: row.Username,
			}
			workspaceSurveyorMap[row.WorkspaceID] = append(workspaceSurveyorMap[row.WorkspaceID], wsItem)
		}
	}

	// 组织工作区到对应项目
	for _, ws := range workspaces {
		projectItem, exists := projectMap[ws.ProjectID]
		if !exists {
			continue
		}

		assigneeInfo := assigneeMap[ws.ID]
		workspaceItem := WorkspaceTreeItem{
			WorkspaceID:   ws.ID,
			WorkspaceName: ws.Name,
			AssigneeID:    assigneeInfo.UserID,
			AssigneeName:  assigneeInfo.Username,
			Surveyors:     workspaceSurveyorMap[ws.ID],
		}

		projectItem.Workspaces = append(projectItem.Workspaces, workspaceItem)
	}

	// 转换为切片
	for _, project := range projectMap {
		projectTree = append(projectTree, *project)
	}

	return projectTree, nil
}

// ================ 工作区仪表板数据 ================

// WorkspaceDashboardResponse 工作区仪表板响应
type WorkspaceDashboardResponse struct {
	WorkspaceID   uint                         `json:"workspace_id"`
	WorkspaceName string                       `json:"workspace_name"`
	ProjectID     uint                         `json:"project_id"`
	ProjectName   string                       `json:"project_name"`
	AssigneeID    uint                         `json:"assignee_id,omitempty"`
	AssigneeName  string                       `json:"assignee_name,omitempty"`
	Tasks         []WorkspaceTaskDashboardItem `json:"tasks"`
	TotalStats    WorkspaceDashboardStats      `json:"total_stats"`
	GeoData       WorkspaceDashboardGeoData    `json:"geo_data"`
}

// WorkspaceTaskDashboardItem 工作区内单个任务的仪表板数据
type WorkspaceTaskDashboardItem struct {
	TaskID     uint          `json:"task_id"`
	UserID     uint          `json:"user_id"`
	Username   string        `json:"username"`
	Status     string        `json:"status"`
	IsRead     bool          `json:"is_read"`
	CreatedAt  time.Time     `json:"created_at"`
	FinishedAt *time.Time    `json:"finished_at,omitempty"`
	PointStats map[int]int64 `json:"point_stats"`
	LastLogin  time.Time     `json:"last_login"`
	LastIP     string        `json:"last_ip"`
}

// WorkspaceDashboardStats 工作区总体统计
type WorkspaceDashboardStats struct {
	TotalTasks       int64         `json:"total_tasks"`
	TotalPoints      int64         `json:"total_points"`
	CompletedTasks   int64         `json:"completed_tasks"`
	PendingTasks     int64         `json:"pending_tasks"`
	PointStatsByType map[int]int64 `json:"point_stats_by_type"`
}

// WorkspaceDashboardGeoData 工作区地理数据
type WorkspaceDashboardGeoData struct {
	PlannedLine  *utils.FeatureCollection `json:"planned_line"`
	PlannedPoint *utils.FeatureCollection `json:"planned_point"`
	ActualLines  []map[string]interface{} `json:"actual_lines"`
	ActualPoints *utils.FeatureCollection `json:"actual_points"`
}

// GetWorkspaceDashboard 获取工作区仪表板数据
func GetWorkspaceDashboard(workspaceID uint, currentUserID uint, currentUserRole string) (*WorkspaceDashboardResponse, error) {
	// 验证工作区访问权限
	var workspace model.Workspace
	if err := config.DB.First(&workspace, workspaceID).Error; err != nil {
		return nil, errors.New("工作区不存在")
	}

	// 权限验证
	if !checkWorkspaceAccess(workspace, currentUserID, currentUserRole) {
		return nil, errors.New("无权访问该工作区")
	}

	// 获取项目信息
	var project model.Project
	if err := config.DB.First(&project, workspace.ProjectID).Error; err != nil {
		return nil, errors.New("项目不存在")
	}

	// 获取工作区负责人信息
	var assignee model.User
	if workspace.AssigneeID > 0 {
		config.DB.First(&assignee, workspace.AssigneeID)
	}

	// 获取工作区下的所有任务
	var tasks []model.Task
	if err := config.DB.Where("workspace_id = ?", workspaceID).Find(&tasks).Error; err != nil {
		return nil, errors.New("查询任务失败: " + err.Error())
	}

	// 构建响应
	resp := &WorkspaceDashboardResponse{
		WorkspaceID:   workspace.ID,
		WorkspaceName: workspace.Name,
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		AssigneeID:    assignee.ID,
		AssigneeName:  assignee.Username,
		Tasks:         []WorkspaceTaskDashboardItem{},
		TotalStats: WorkspaceDashboardStats{
			PointStatsByType: make(map[int]int64),
		},
		GeoData: WorkspaceDashboardGeoData{
			PlannedLine:  &utils.FeatureCollection{Type: "FeatureCollection", Features: []utils.Feature{}},
			PlannedPoint: &utils.FeatureCollection{Type: "FeatureCollection", Features: []utils.Feature{}},
			ActualLines:  []map[string]interface{}{},
			ActualPoints: &utils.FeatureCollection{Type: "FeatureCollection", Features: []utils.Feature{}},
		},
	}

	// 处理每个任务
	taskIDs := make([]uint, 0, len(tasks))
	userIDs := make([]uint, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
		userIDs = append(userIDs, task.AssigneeID)
	}

	// 批量获取用户信息
	var users []model.User
	if len(userIDs) > 0 {
		config.DB.Where("id IN (?)", userIDs).Find(&users)
	}
	userMap := make(map[uint]model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	// 批量获取点位统计
	taskPointStats := make(map[uint]map[int]int64)
	if len(taskIDs) > 0 {
		type PointStatResult struct {
			TaskID uint
			Type   int
			Count  int64
		}
		var stats []PointStatResult
		config.DB.Table("points").
			Select("task_id, type, count(*) as count").
			Where("task_id IN (?)", taskIDs).
			Group("task_id, type").
			Scan(&stats)

		for _, s := range stats {
			if taskPointStats[s.TaskID] == nil {
				taskPointStats[s.TaskID] = make(map[int]int64)
			}
			taskPointStats[s.TaskID][s.Type] = s.Count
		}
	}

	// 处理每个任务
	for _, task := range tasks {
		user, userExists := userMap[task.AssigneeID]
		if !userExists {
			continue
		}

		taskItem := WorkspaceTaskDashboardItem{
			TaskID:     task.ID,
			UserID:     user.ID,
			Username:   user.Username,
			Status:     task.Status,
			IsRead:     task.IsRead,
			CreatedAt:  task.CreatedAt,
			LastLogin:  user.LastLoginDate,
			LastIP:     user.LastIP,
			PointStats: taskPointStats[task.ID],
		}

		if task.FinishedAt != nil {
			taskItem.FinishedAt = task.FinishedAt
		}

		resp.Tasks = append(resp.Tasks, taskItem)

		// 更新总体统计
		resp.TotalStats.TotalTasks++
		if task.Status == "completed" {
			resp.TotalStats.CompletedTasks++
		} else {
			resp.TotalStats.PendingTasks++
		}

		// 更新点位类型统计
		for pointType, count := range taskPointStats[task.ID] {
			resp.TotalStats.TotalPoints += count
			resp.TotalStats.PointStatsByType[pointType] += count
		}
	}

	// 获取地理数据（使用第一个任务的规划数据，假设同一工作区下任务规划数据相同）
	if len(tasks) > 0 {
		firstTask := tasks[0]
		resp.GeoData.PlannedLine = utils.SingleShpToGeoJSON(firstTask.PlannedLineShpUrl)
		resp.GeoData.PlannedPoint = utils.SingleShpToGeoJSON(firstTask.PlannedPointShpUrl)
	}

	// 获取所有实际轨迹
	var actualRoutes []model.ActualRoute
	if len(taskIDs) > 0 {
		config.DB.Where("task_id IN (?)", taskIDs).Find(&actualRoutes)
	}

	var actualGeoms []map[string]interface{}
	for _, route := range actualRoutes {
		if route.ActualLineGeom == "" {
			continue
		}
		var featureObj map[string]interface{}
		err := json.Unmarshal([]byte(route.ActualLineGeom), &featureObj)
		if err == nil {
			actualGeoms = append(actualGeoms, featureObj)
		}
	}
	resp.GeoData.ActualLines = actualGeoms

	// 获取所有实际调查点
	if len(taskIDs) > 0 {
		type PointRecord struct {
			ID          uint
			GeomJson    string
			PathID      string
			Type        int
			PointSerial string
		}
		var ptRecords []PointRecord
		config.DB.Raw("SELECT id, point_serial, ST_AsGeoJSON(geom) as geom_json, path_id, type FROM points WHERE task_id IN (?)", taskIDs).Scan(&ptRecords)

		for _, pt := range ptRecords {
			var geom utils.Geometry
			err := json.Unmarshal([]byte(pt.GeomJson), &geom)
			if err != nil {
				continue
			}

			props := make(map[string]interface{})
			props["point_id"] = pt.ID
			props["point_serial"] = pt.PointSerial
			props["_path_id"] = pt.PathID
			props["_type"] = pt.Type

			feat := utils.Feature{
				Type:       "Feature",
				Geometry:   geom,
				Properties: props,
			}
			resp.GeoData.ActualPoints.Features = append(resp.GeoData.ActualPoints.Features, feat)
		}
	}

	return resp, nil
}

// checkWorkspaceAccess 检查工作区访问权限
func checkWorkspaceAccess(workspace model.Workspace, currentUserID uint, currentUserRole string) bool {
	switch currentUserRole {
	case "first_admin":
		return true // 一级管理员可以访问所有工作区
	case "sec_admin":
		// 二级管理员只能访问自己创建的项目下的工作区
		var project model.Project
		if err := config.DB.First(&project, workspace.ProjectID).Error; err != nil {
			return false
		}
		return project.CreatorID == currentUserID
	case "third_admin":
		// 三级管理员只能访问自己负责的工作区
		return workspace.AssigneeID == currentUserID
	default:
		return false
	}
}

// ================ 带权限过滤的点位分页查询 ================

// GetPaginatedPointsForAdmin 带权限过滤的点位分页查询
func GetPaginatedPointsForAdmin(page, pageSize int, username, dateStr, typeStr string,
	currentUserID uint, currentUserRole string, projectID, workspaceID, taskID, userID uint) ([]dto.PointListItem, int64, error) {

	// 构建基础查询
	query := config.DB.Table("points").
		Select("points.id as point_id, points.task_id, points.path_id, points.type, points.point_serial, points.created_at, users.username").
		Joins("left join users on points.user_id = users.id")

	// 应用权限过滤
	query = applyPointsAccessControl(query, currentUserID, currentUserRole)

	// 应用筛选条件
	if username != "" {
		query = query.Where("users.username LIKE ?", "%"+username+"%")
	}

	if dateStr != "" {
		startOfDay, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err == nil {
			endOfDay := startOfDay.Add(24 * time.Hour)
			query = query.Where("points.created_at >= ? AND points.created_at < ?", startOfDay, endOfDay)
		}
	}

	if typeStr != "" {
		query = query.Where("points.type = ?", typeStr)
	}

	// 应用额外的筛选参数
	if projectID > 0 {
		// 通过任务和工作区关联到项目
		query = query.Joins("JOIN tasks ON points.task_id = tasks.id").
			Joins("JOIN workspaces ON tasks.workspace_id = workspaces.id").
			Where("workspaces.project_id = ?", projectID)
	}

	if workspaceID > 0 {
		query = query.Joins("JOIN tasks ON points.task_id = tasks.id").
			Where("tasks.workspace_id = ?", workspaceID)
	}

	if taskID > 0 {
		query = query.Where("points.task_id = ?", taskID)
	}

	if userID > 0 {
		query = query.Where("points.user_id = ?", userID)
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 分页查询
	var items []dto.PointListItem
	offset := (page - 1) * pageSize
	err := query.Order("points.created_at desc").Offset(offset).Limit(pageSize).Scan(&items).Error

	return items, total, err
}

// applyPointsAccessControl 应用点位访问权限控制
func applyPointsAccessControl(query *gorm.DB, currentUserID uint, currentUserRole string) *gorm.DB {
	switch currentUserRole {
	case "first_admin":
		// 一级管理员可以查看所有点位
		return query
	case "sec_admin":
		// 二级管理员只能查看自己创建的项目下的点位
		return query.Joins("JOIN tasks ON points.task_id = tasks.id").
			Joins("JOIN workspaces ON tasks.workspace_id = workspaces.id").
			Joins("JOIN projects ON workspaces.project_id = projects.id").
			Where("projects.creator_id = ?", currentUserID)
	case "third_admin":
		// 三级管理员只能查看自己负责的工作区下的点位
		return query.Joins("JOIN tasks ON points.task_id = tasks.id").
			Joins("JOIN workspaces ON tasks.workspace_id = workspaces.id").
			Where("workspaces.assignee_id = ?", currentUserID)
	default:
		// 调查员只能查看自己的点位（通过其他接口）
		return query.Where("points.user_id = ?", currentUserID)
	}
}

// ================ 工作区地理数据 GeoJSON 接口 ================

// GetWorkspaceGeoJSONByProjectID 根据项目ID获取下属所有工作区的GeoJSON
func GetWorkspaceGeoJSONByProjectID(projectID uint, shpPath string) (*utils.FeatureCollection, error) {
	// 查询该项目下的所有工作区名称（三级名）
	var workspaces []model.Workspace
	if err := config.DB.Where("project_id = ?", projectID).Find(&workspaces).Error; err != nil {
		return nil, errors.New("查询工作区失败: " + err.Error())
	}
	if len(workspaces) == 0 {
		return &utils.FeatureCollection{Type: "FeatureCollection", Features: []utils.Feature{}}, nil
	}

	// 收集三级名列表
	var thirdNames []string
	for _, ws := range workspaces {
		thirdNames = append(thirdNames, ws.Name)
	}

	// 从SHP文件中过滤出匹配的要素
	return filterGeoJSONByThirdNames(shpPath, thirdNames)
}

// GetWorkspaceGeoJSONByWorkspaceID 根据工作区ID获取对应工作区的GeoJSON
func GetWorkspaceGeoJSONByWorkspaceID(workspaceID uint, shpPath string) (*utils.FeatureCollection, error) {
	// 查询工作区名称（三级名）
	var workspace model.Workspace
	if err := config.DB.First(&workspace, workspaceID).Error; err != nil {
		return nil, errors.New("工作区不存在: " + err.Error())
	}

	// 从SHP文件中过滤出匹配的要素
	return filterGeoJSONByThirdNames(shpPath, []string{workspace.Name})
}

// filterGeoJSONByThirdNames 从SHP文件中过滤出指定三级名的要素
func filterGeoJSONByThirdNames(shpPath string, thirdNames []string) (*utils.FeatureCollection, error) {
	// 将三级名列表转换为map便于快速查找
	nameMap := make(map[string]bool)
	for _, name := range thirdNames {
		nameMap[name] = true
	}

	// 读取整个SHP文件的GeoJSON
	fullCollection := utils.SingleShpToGeoJSON(shpPath)
	if fullCollection == nil {
		return &utils.FeatureCollection{Type: "FeatureCollection", Features: []utils.Feature{}}, nil
	}

	// 过滤要素
	filteredFeatures := []utils.Feature{}
	for _, feature := range fullCollection.Features {
		if thirdName, ok := feature.Properties["三级名"].(string); ok {
			if nameMap[thirdName] {
				filteredFeatures = append(filteredFeatures, feature)
			}
		}
	}

	return &utils.FeatureCollection{
		Type:     "FeatureCollection",
		Features: filteredFeatures,
	}, nil
}
