package models

import (
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"strings"
	"time"
)

var ErrProjectExists = errors.New("项目已存在")

type Project struct {
	ID        int            `gorm:"primaryKey"                               json:"id"`
	UserID    int            `gorm:"index;not null;uniqueIndex:ux_user_name_alive,priority:1" json:"user_id"`
	Name      string         `gorm:"size:128;not null;uniqueIndex:ux_user_name_alive,priority:2" json:"name"`
	Color     string         `gorm:"size:16"                                   json:"color,omitempty"`
	SortOrder int64          `gorm:"not null;default:0"                        json:"sort_order"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                     json:"-"`
	Alive     uint8          `gorm:"->;type:TINYINT(1) GENERATED ALWAYS AS (IF(deleted_at IS NULL,1,0)) STORED;uniqueIndex:ux_user_name_alive,priority:3" json:"-"`

	User User `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

func (t *Project) BeforeCreate(tx *gorm.DB) error {
	if t.SortOrder == 0 {
		t.SortOrder = time.Now().UnixNano()
	}
	return nil
}

func AddProject(project Project) (Project, error) {

	if err := d.Db.Create(&project).Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return Project{}, ErrProjectExists
		}
		return Project{}, err
	}
	return project, nil
}

func ProjectList(userID int, page, size int) ([]Project, int64, error) {
	var (
		items []Project
		total int64
	)
	q := d.Db.Model(&Project{}).Where("user_id = ?", userID)

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := q.Order("sort_order DESC, id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&items).Error

	return items, total, err
}

func DeleteProjectAndTasks(projectID, userID int) (projAffected int64, taskAffected int64, err error) {
	err = d.Db.Transaction(func(tx *gorm.DB) error {

		res := tx.Where("id = ? AND user_id = ?", projectID, userID).Delete(&Project{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		projAffected = res.RowsAffected

		res2 := tx.Where("user_id = ? AND project_id = ?", userID, projectID).Delete(&Task{})
		if res2.Error != nil {
			return res2.Error
		}
		taskAffected = res2.RowsAffected

		return nil
	})
	return
}

func GetProjectInfoByIDAndUserID(id int, userid int) (Project, error) {
	var project Project
	err := d.Db.Where("id = ? And user_id = ?", id, userid).First(&project).Error
	return project, err
}

func UpdateProjectByIDAndUserID(update map[string]interface{}, id int, userid int) (Project, error, int64) {
	var project Project
	res := d.Db.Where("id = ? And user_id = ?", id, userid).Updates(update)
	if err := res.Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return Project{}, ErrProjectExists, 0
		}
		return Project{}, err, 0
	}
	if err := d.Db.First(&project, "id = ? And user_id = ?", id, userid).Error; err != nil {
		return project, err, 0
	}
	return project, nil, res.RowsAffected

}

func GetProjectListByUserIDAndName(UserID int, name string, page, size int) ([]Project, int64, error) {
	var (
		items []Project
		total int64
	)
	db := d.Db.Model(&Project{}).Where("user_id = ?", UserID)
	name = strings.TrimSpace(name)
	if name != "" {
		db = db.Where("name LIKE ?", "%"+name+"%")
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("sort_order DESC, id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&items).Error

	return items, total, err
}
