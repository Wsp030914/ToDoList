package models

import (
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
	Alive        uint8  `gorm:"->;type:TINYINT(1) GENERATED ALWAYS AS (IF(deleted_at IS NULL,1,0)) STORED;uniqueIndex:ux_user_username_alive,priority:2;uniqueIndex:ux_user_email_alive,priority:2" json:"-"`
}

func GetUserInfoByUsername(username string) (User, error) {
	var user User
	err := d.Db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, nil
	}
	return user, err
}

func GetUserInfoByID(UID int) (User, error) {
	var user User
	err := d.Db.Where("ID = ?", UID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, nil
	}
	return user, err
}

func AddUser(user User) (User, error) {

	if err := d.Db.Create(&user).Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return User{}, ErrUserExists
		}
		return User{}, err
	}
	return user, nil
}

func GetVersionByID(uid int) (User, error) {
	var u User
	err := d.Db.Select("id, token_version").
		Where("id = ?", uid).
		First(&u).Error

	return u, err
}

func GetUserInfoByEmail(email string) (User, error) {
	var user User
	err := d.Db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, nil
	}
	return user, err
}

func UpdateUser(update map[string]interface{}, uid int) (User, error, int64) {
	var user User
	res := d.Db.Model(&User{}).Where("id = ? ", uid).Updates(update)
	if err := res.Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return User{}, ErrUserExists, 0
		}
		return User{}, err, 0
	}
	if err := d.Db.First(&user, "id = ?", uid).Error; err != nil {
		return user, err, 0
	}
	return user, nil, res.RowsAffected
}
