package models

import (
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

type Task struct {
	ID            int            `gorm:"primaryKey"                           json:"id"`
	UserID        int            `gorm:"not null;index:idx_user_sort,priority:1;index:idx_user_proj_sort,priority:1;uniqueIndex:ux_task_user_proj_title_alive,priority:1" json:"user_id"`
	ProjectID     *int           `gorm:"index;index:idx_user_proj_sort,priority:2"                                   json:"project_id,omitempty"`
	Title         string         `gorm:"size:200;not null;uniqueIndex:ux_task_user_proj_title_alive,priority:3" json:"title"`
	ProjectIDNorm uint64         `gorm:"->;type:BIGINT UNSIGNED GENERATED ALWAYS AS (COALESCE(project_id, 0)) STORED;uniqueIndex:ux_task_user_proj_title_alive,priority:2" json:"-"`
	ContentMD     string         `gorm:"type:longtext"                         json:"content_md"`
	Status        string         `gorm:"type:enum('todo','doing','done','archived');not null;default:'todo'" json:"status"`
	Priority      int            `gorm:"type:tinyint;not null;default:3"       json:"priority"`
	SortOrder     int64          `gorm:"not null;default:0;index:idx_user_sort,priority:2;index:idx_user_proj_sort,priority:3" json:"sort_order"`
	StartAt       *time.Time     `json:"start_at"`
	DueAt         *time.Time     `json:"due_at"`
	DoneAt        *time.Time     `json:"done_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Alive         uint8          `gorm:"->;type:TINYINT(1) GENERATED ALWAYS AS (IF(deleted_at IS NULL,1,0)) STORED;uniqueIndex:ux_task_user_proj_title_alive,priority:4" json:"-"`
	ContentHtml   string         `gorm:"type:longtext"                         json:"content_html"`
	User          User           `gorm:"foreignKey:UserID;references:ID"     json:"-"`
	Project       *Project       `gorm:"foreignKey:ProjectID;references:ID"  json:"-"`
}

func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.SortOrder == 0 {
		t.SortOrder = time.Now().UnixNano()
	}
	return nil
}

const (
	TaskTodo     = "todo"
	TaskDoing    = "doing"
	TaskDone     = "done"
	TaskArchived = "archived"
)

var ErrTaskExists = errors.New("任务已存在")

func GetTaskByUserProjectTitle(uid int, pid *int, title string) (Task, error) {
	var task Task
	err := d.Db.Where("user_id = ? AND project_id <=> ? AND title = ?", uid, pid, title).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Task{}, nil
	}
	return task, err
}

func CreateTaskByUidAndTask(uid int, t Task) (Task, error) {
	t.UserID = uid
	t.ID = 0

	if err := d.Db.Create(&t).Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return Task{}, ErrTaskExists
		}
		return Task{}, err
	}
	return t, nil
}

func GetProjectByID(uid int, pid int) (Project, error) {

	var project Project
	err := d.Db.Where("user_id = ? AND id = ?", uid, pid).First(&project).Error
	return project, err
}

func TaskList(uid int, pid *int, page int, size int, status string) ([]Task, int64, error) {
	var (
		task  []Task
		total int64
		tx    *gorm.DB
	)
	tx = d.Db.Model(&Task{}).Where("user_id = ? And project_id <=> ? And status = ?", uid, pid, status)
	if status == "" {
		tx = d.Db.Model(&Task{}).Where("user_id = ? And project_id <=> ? ", uid, pid)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := tx.Order("sort_order DESC, priority DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&task).Error

	return task, total, err
}

func DeleteByIDAndProjectIDAndUID(id int, pid *int, uid int) (error, int64) {
	res := d.Db.Where("user_id = ? And project_id <=> ? And id = ? ", uid, pid, id).Delete(&Task{})
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound, 0
	}
	return res.Error, res.RowsAffected
}

func DeleteByStatus(uid int, pid *int, status string) (error, int64) {
	res := d.Db.Where("user_id = ? And project_id <=> ? And status = ? ", uid, pid, status).Delete(&Task{})
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound, 0
	}
	return res.Error, res.RowsAffected

}

// models/task.go

func GetTaskByIDAndUID(id int, uid int) (Task, error) {
	var t Task
	err := d.Db.Where("id = ? AND user_id = ?", id, uid).First(&t).Error

	return t, err
}

func UpdateTaskByIDAndUID(update map[string]interface{}, id int, uid int) (Task, error, int64) {
	var t Task
	res := d.Db.Model(&Task{}).Where("id = ? AND user_id = ?", id, uid).Updates(update)
	if err := res.Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return Task{}, ErrTaskExists, 0
		}
		return Task{}, err, 0
	}
	if err := d.Db.Where("id = ? AND user_id = ?", id, uid).First(&t).Error; err != nil {
		return t, err, 0
	}
	return t, nil, res.RowsAffected
}
