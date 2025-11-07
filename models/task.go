package models

import (
	"NewStudent/dao"
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
	ContentMD     string         `gorm:"type:longtext"                         json:"content_md"` // Markdown 正文
	Status        string         `gorm:"type:enum('todo','doing','done','archived');not null;default:'todo'" json:"status"`
	Priority      int            `gorm:"type:tinyint;not null;default:3"       json:"priority"` // 1(最高)…5(最低)
	SortOrder     int64          `gorm:"not null;default:0;index:idx_user_sort,priority:2;index:idx_user_proj_sort,priority:3" json:"sort_order"`
	StartAt       *time.Time     `json:"start_at"`
	DueAt         *time.Time     `json:"due_at"`
	DoneAt        *time.Time     `json:"done_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Alive         uint8          `gorm:"->;type:TINYINT(1) GENERATED ALWAYS AS (IF(deleted_at IS NULL,1,0)) STORED;uniqueIndex:ux_task_user_proj_title_alive,priority:4" json:"-"`

	User    User     `gorm:"foreignKey:UserID;references:ID"     json:"-"`
	Project *Project `gorm:"foreignKey:ProjectID;references:ID"  json:"-"`
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
	err := dao.Db.Where("user_id = ? AND project_id <=> ? AND title = ?", uid, pid, title).First(&task).Error

	return task, err
}

func CreateTaskByUidAndTask(uid int, t Task) (Task, error) {
	t.UserID = uid
	t.ID = 0

	if err := dao.Db.Create(&t).Error; err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return Task{}, ErrTaskExists
		}
		return Task{}, err
	}
	return t, nil
}

func GetProjectByID(uid int, pid *int) (Project, error) {
	var project Project
	err := dao.Db.Where("user_id = ? AND project_id <=> ?", uid, pid).First(&project).Error

	return project, err
}
