package service

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
	"errors"
	"mime/multipart"

	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"
)

// ImportUsersFromExcel 导入用户
// 规则：二管(sec_admin)导入的是三管(third_admin)；三管(third_admin)导入的是野外调查员(user)
func ImportUsersFromExcel(file multipart.File, creatorID uint, creatorRole string) (int, error) {
	// 判断要创建的目标角色
	var targetRole string
	if creatorRole == "sec_admin" {
		targetRole = "third_admin"
	} else if creatorRole == "third_admin" {
		targetRole = "user"
	} else {
		return 0, errors.New("当前角色无权导入用户")
	}

	// 读取 Excel 文件
	f, err := excelize.OpenReader(file)
	if err != nil {
		return 0, errors.New("无法读取Excel文件: " + err.Error())
	}
	defer f.Close()

	// 假设数据在第一张表 "Sheet1"
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return 0, errors.New("无法读取Sheet1数据: " + err.Error())
	}

	var users []model.User
	successCount := 0

	// 假设 Excel 的列顺序为: 0:Username, 1:Email, 2:Password (第一行是表头)
	for i, row := range rows {
		if i == 0 {
			continue // 跳过表头
		}
		if len(row) < 3 {
			continue // 数据不全跳过
		}

		username := row[0]
		email := row[1]
		plainPassword := row[2]

		// 对明文密码进行加密
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)

		user := model.User{
			Username:  username,
			Email:     email,
			Password:  string(hashedPassword),
			Role:      targetRole,
			CreatorID: creatorID,
		}
		users = append(users, user)
	}

	// 批量插入数据库 (注: 若遇到同名冲突可能会报错，生产环境可优化为跳过已存在记录)
	if len(users) > 0 {
		if err := config.DB.Create(&users).Error; err != nil {
			return 0, errors.New("存入数据库失败(可能存在用户名重复): " + err.Error())
		}
		successCount = len(users)
	}

	return successCount, nil
}

// GetManagedUsers 查询其管辖的用户 (支持按用户名模糊搜索)
func GetManagedUsers(creatorID uint, searchName string) ([]model.User, error) {
	var users []model.User
	query := config.DB.Where("creator_id = ?", creatorID)

	if searchName != "" {
		// 模糊查询
		query = query.Where("username LIKE ?", "%"+searchName+"%")
	}

	// 隐藏密码返回
	err := query.Select("id, username, email, role, creator_id, created_date, last_login_date, last_ip").Find(&users).Error
	return users, err
}

// UpdateManagedUser 更新下属用户信息 (包括重置密码)
func UpdateManagedUser(creatorID uint, targetUserID uint, newUsername, newEmail, newPassword string) error {
	// 1. 先验证该 targetUser 是否真的属于该 creator
	var user model.User
	if err := config.DB.Where("id = ? AND creator_id = ?", targetUserID, creatorID).First(&user).Error; err != nil {
		return errors.New("目标用户不存在或您无权修改")
	}

	// 2. 构建更新内容
	updates := make(map[string]interface{})
	if newUsername != "" {
		updates["username"] = newUsername
	}
	if newEmail != "" {
		updates["email"] = newEmail
	}
	if newPassword != "" { // 如果上传了新密码，则进行加密后更新
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		updates["password"] = string(hashedPassword)
	}

	return config.DB.Model(&user).Updates(updates).Error
}

// DeleteManagedUser 删除下属用户
func DeleteManagedUser(creatorID uint, targetUserID uint) error {
	// GORM 默认是软删除（如果包含 DeletedAt 字段），这里假设直接硬删除
	res := config.DB.Where("id = ? AND creator_id = ?", targetUserID, creatorID).Delete(&model.User{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("目标用户不存在或您无权删除")
	}
	return nil
}
