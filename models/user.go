package models

import (
	"NewStudent/dao"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID        int            `gorm:"primaryKey"                 json:"id"`
	Email     string         `gorm:"size:255;"                  json:"email"`
	Password  string         `gorm:"size:255;not null"          json:"-"`
	Username  string         `gorm:"size:64;not null"           json:"username"`
	AvatarURL string         `gorm:"size:512"                   json:"avatar_url"`
	Timezone  string         `gorm:"size:64;default:Asia/Shanghai" json:"timezone"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                      json:"-"`
}

func GetUserInfoByUsername(username string) (User, error) {
	var user User
	err := dao.Db.Where("username = ?", username).First(&user).Error
	return user, err
}

func AddUser(username string, password string) (int, error) {
	// ① 用 bcrypt 生成密码哈希
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	// ② 入库时保存哈希而非明文
	user := User{
		Username: username,
		Password: string(hash),
	}
	err = dao.Db.Create(&user).Error
	return user.ID, err
}
