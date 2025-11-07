package models

import (
	"NewStudent/dao"
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

var ErrUserExists = errors.New("用户已存在")

type User struct {
	ID           int            `gorm:"primaryKey"                 json:"id"`
	Email        string         `gorm:"size:255;not null;"                  json:"email"`
	Password     string         `gorm:"size:255;not null"          json:"-"`
	Username     string         `gorm:"size:64;not null;"           json:"username"`
	AvatarURL    string         `gorm:"size:512"                   json:"avatar_url"`
	Timezone     string         `gorm:"size:64;default:Asia/Shanghai" json:"timezone"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index"                      json:"-"`
	TokenVersion int            `gorm:"not null;default:1"  json:"-"`

	UsernameNorm string `gorm:"->;type:VARCHAR(64)  GENERATED ALWAYS AS (LOWER(username)) STORED;uniqueIndex:ux_user_username_alive,priority:1" json:"-"`
	EmailNorm    string `gorm:"->;type:VARCHAR(255) GENERATED ALWAYS AS (LOWER(email))    STORED;uniqueIndex:ux_user_email_alive,priority:1" json:"-"`
	// 软删“活跃”标志，同列可同时参与两条唯一索引
	Alive uint8 `gorm:"->;type:TINYINT(1) GENERATED ALWAYS AS (IF(deleted_at IS NULL,1,0)) STORED;uniqueIndex:ux_user_username_alive,priority:2;uniqueIndex:ux_user_email_alive,priority:2" json:"-"`
}

func GetUserInfoByUsername(username string) (User, error) {
	var user User
	err := dao.Db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, nil
	}
	return user, err
}

func AddUser(user User) (User, error) {

	if err := dao.Db.Create(&user).Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return User{}, ErrUserExists
		}
		return User{}, err
	}
	return user, nil
}

func GetUserInfoByEmail(email string) (User, error) {
	var user User
	err := dao.Db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, nil
	}
	return user, err
}

func LogoutUser(uid int) (error, int64) {
	res := dao.Db.Model(&User{}).
		Where("id = ?", uid).
		UpdateColumn("token_version", gorm.Expr("token_version + 1"))
	return res.Error, res.RowsAffected
}

func UpdateUser(update map[string]interface{}, uid int) (User, error, int64) {
	var user User
	res := dao.Db.Where("id = ? ", uid).Updates(update)
	if err := res.Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return User{}, ErrUserExists, 0
		}
		return User{}, err, 0
	}
	if err := dao.Db.First(&user, "id = ?", uid).Error; err != nil {
		return user, err, 0
	}
	return user, nil, res.RowsAffected
}
