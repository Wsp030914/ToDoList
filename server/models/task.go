package models

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type Task struct {
	ID          int        `gorm:"primaryKey"                           json:"id"`
	UserID      int        `gorm:"not null;index:idx_user_sort,priority:1;index:idx_user_proj_sort,priority:1;uniqueIndex:ux_task_user_proj_title,priority:1" json:"user_id"`
	ProjectID   int        `gorm:"not null;index;index:idx_user_proj_sort,priority:2;uniqueIndex:ux_task_user_proj_title,priority:2"                                   json:"project_id"`
	Title       string     `gorm:"size:200;not null;uniqueIndex:ux_task_user_proj_title,priority:3" json:"title"`
	ContentMD   string     `gorm:"type:longtext"                         json:"content_md"`
	Status      string     `gorm:"type:enum('todo','done');not null;default:'todo';index:idx_tasks_due_watch,priority:1" json:"status"`
	Priority    int        `gorm:"type:tinyint;not null;default:3"       json:"priority"`
	SortOrder   int64      `gorm:"not null;default:0;index:idx_user_sort,priority:2;index:idx_user_proj_sort,priority:3" json:"sort_order"`
	DueAt       *time.Time `gorm:"index:idx_tasks_due_watch,priority:2" json:"due_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ContentHtml string     `gorm:"type:longtext"                         json:"content_html"`
	Notified    bool       `gorm:"not null;default:false;index:idx_tasks_due_watch,priority:3"`
}

func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if t.SortOrder == 0 {
		t.SortOrder = time.Now().UnixNano()
	}
	return nil
}

const (
	TaskTodo = "todo"
	TaskDone = "done"
)

var ErrTaskExists = errors.New("任务已存在")

func GetTaskByUserProjectTitle(uid int, pid int, title string) (Task, error) {
	var task Task
	err := d.Db.Where("user_id = ? AND project_id = ? AND title = ?", uid, pid, title).First(&task).Error
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

func TaskListAll(uid int, pid int, status string) ([]Task, int64, error) {
	var (
		task  []Task
		total int64
		tx    *gorm.DB
	)

	tx = d.Db.Model(&Task{}).Where("user_id = ? AND project_id = ?", uid, pid)
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := tx.Order("sort_order DESC, priority DESC").
		Find(&task).Error

	if err != nil {
		return nil, 0, err
	}
	return task, total, nil
}

func DeleteByIDAndProjectIDAndUID(id int, pid int, uid int) (int64, error) {
	res := d.Db.Where("user_id = ? And project_id = ? And id = ? ", uid, pid, id).Delete(&Task{})
	if res.RowsAffected == 0 {
		return 0, gorm.ErrRecordNotFound
	}
	return res.RowsAffected, res.Error
}

func GetTaskByIDAndProjectIDAndUID(id int, uid int, pid int) (Task, error) {
	var t Task
	err := d.Db.Where("id = ? AND user_id = ? And project_id = ?", id, uid, pid).First(&t).Error

	return t, err
}

func FindDueTasks(ctx context.Context, from, to time.Time, limit int) ([]Task, error) {
	var tasks []Task
	err := d.Db.WithContext(ctx).Where("status = ? AND notified = ? AND due_at IS NOT NULL AND due_at >= ? AND due_at < ?",
		"todo", false, from, to).
		Order("due_at ASC").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func UpdatedDueTasks(ctx context.Context, id int) (int64, error) {
	 res := d.Db.WithContext(ctx).
        Model(&Task{}).
        Where("id = ? AND notified = ?", id, false).
        Update("notified", true)
    return res.RowsAffected, res.Error
}

func UpdateTaskByIDAndUID(update map[string]interface{}, id int, uid int) (Task, int64, error) {
	var t Task
	res := d.Db.Model(&Task{}).Where("id = ? AND user_id = ?", id, uid).Updates(update)
	if err := res.Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return Task{}, 0, ErrTaskExists
		}
		return Task{}, 0, err
	}
	if err := d.Db.Where("id = ? AND user_id = ?", id, uid).First(&t).Error; err != nil {
		return t, 0, err
	}
	return t, res.RowsAffected, nil
}
