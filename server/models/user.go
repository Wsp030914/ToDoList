package models

import (
	"context"
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

var ErrUserExists = errors.New("用户已存在")

type User struct {
	ID           int            `gorm:"primaryKey"                 json:"id"`
	Email        string         `gorm:"size:255;not null;uniqueIndex"                  json:"email"`
	Password     string         `gorm:"size:255;not null"          json:"-"`
	Username     string         `gorm:"size:64;not null;uniqueIndex"           json:"username"`
	AvatarURL    string         `gorm:"size:512"                   json:"avatar_url"`
	Timezone     string         `gorm:"size:64;default:Asia/Shanghai" json:"timezone"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	TokenVersion int            `gorm:"not null;default:1"  json:"-"`

}

func GetUserInfoByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := d.Db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, nil
	}
	return user, err
}

func GetUserInfoByID(ctx context.Context, UID int) (User, error) {
	var user User
	err := d.Db.WithContext(ctx).Where("ID = ?", UID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, nil
	}
	return user, err
}

func AddUser(ctx context.Context, user User) (User, error) {

	if err := d.Db.WithContext(ctx).Create(&user).Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return User{}, ErrUserExists
		}
		return User{}, err
	}
	return user, nil
}

func GetVersionByID(ctx context.Context, uid int) (User, error) {
	var u User
	err := d.Db.WithContext(ctx).Select("id, token_version").
		Where("id = ?", uid).
		First(&u).Error

	return u, err
}

func GetUserInfoByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := d.Db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, nil
	}
	return user, err
}

func UpdateUser(ctx context.Context, update map[string]interface{}, uid int) (User, error, int64) {
	var user User
	res := d.Db.WithContext(ctx).Model(&User{}).Where("id = ? ", uid).Updates(update)
	if err := res.Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return User{}, ErrUserExists, 0
		}
		return User{}, err, 0
	}
	if err := d.Db.WithContext(ctx).First(&user, "id = ?", uid).Error; err != nil {
		return user, err, 0
	}
	return user, nil, res.RowsAffected
}
