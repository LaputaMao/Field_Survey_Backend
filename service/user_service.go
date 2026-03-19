package service

import (
	"Field_Survey_Backend/config"
	"Field_Survey_Backend/model"
	"errors"
	"time"

	"gorm.io/gorm"
)

func Login(username, password, ip string) (*model.User, error) {
	var user model.User

	// 1. 查询用户是否存在
	err := config.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}

	// 2. 校验权限限制：App 端只允许角色为 "user" 的人登录
	if user.Role != "user" {
		return nil, errors.New("权限不足，管理员请使用网页端后台登录")
	}

	//// 3. 校验密码 (bcrypt 加密)
	//err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	//if err != nil {
	//	return nil, errors.New("用户名或密码错误")
	//}
	// 3. 校验密码 明文对比，仅供测试
	if user.Password != password {
		return nil, errors.New("用户名或密码错误")
	}
	// 4. 登录成功，更新最后登录时间和IP
	config.DB.Model(&user).Updates(model.User{
		LastLoginDate: time.Now(),
		LastIP:        ip,
	})

	return &user, nil
}
