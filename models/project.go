package models

import (
	"NewStudent/dao"
	"gorm.io/gorm"
	"time"
)

type Project struct {
	ID        int            `gorm:"primaryKey"            json:"id"`
	UserID    int            `gorm:"index;not null"        json:"user_id"`
	Name      string         `gorm:"size:128;not null"     json:"name"`
	Color     *string        `gorm:"size:16"               json:"color,omitempty"`
	SortOrder int64          `gorm:"not null;default:0"    json:"sort_order"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                 json:"-"`

	// 关联（可选）
	User User `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

func GetProjectInfoByNameAndUserID(name string, userid int) (Project, error) {
	var project Project
	err := dao.Db.Where("user_id = ? AND name = ?", userid, name).First(&project).Error
	return project, err
}

func AddProject(name string, userid int) (Project, error) {
	project := Project{
		Name:      name,
		UserID:    userid,
		SortOrder: time.Now().UnixNano(),
	}
	err := dao.Db.Create(&project).Error
	return project, err
}

func ProjectList(userID int, page, size int) ([]Project, int64, error) {
	var (
		items []Project
		total int64
	)
	q := dao.Db.Model(&Project{}).Where("user_id = ?", userID)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := q.Order("sort_order DESC, id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&items).Error

	return items, total, err
}

func DeleteById(id int, userid int) (int64, error) {

	res := dao.Db.Where("id = ? And user_id = ?", id, userid).Delete(&Project{})
	return res.RowsAffected, res.Error
}

func GetProjectInfoById(id int, userid int) (Project, error) {
	var project Project
	err := dao.Db.Where("id = ? And user_id = ?", id, userid).First(&project).Error
	return project, err
}
