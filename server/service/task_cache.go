package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type TaskListCache struct {
	Items []TaskSummary 	`json:"items"`
	Total int64            `json:"total"`
}

func taskKey(uid, id int) string {
	return fmt.Sprintf("task:detail:%d:%d", uid, id)
}

func taskListKey(uid, pid int, status string) string {
	if status == "" {
		status = "all"
	}
	return fmt.Sprintf("task:list:%d:%d:%s", uid, pid, status)
}

func SetaskDetailCache(ctx context.Context, td *TaskDetail) error {
	key := taskKey(td.UserID, td.ID)
	b, err := json.Marshal(td)
	if err != nil {
		return err
	}

	return c.Rdb.Set(ctx, key, b, time.Hour).Err()
}

func GetTaskDetailCache(ctx context.Context, uid, id int) (*TaskDetail, error) {
	key := taskKey(uid, id)
	data, err := c.Rdb.Get(ctx, key).Bytes()
	if err == nil {
		var td TaskDetail
		if uerr := json.Unmarshal(data, &td); uerr != nil {
			return nil, uerr
		}
		return  &td, nil
	}
	return nil, err
}

func DelTaskDetailCache(ctx context.Context, uid, id int) error {
	key := taskKey(uid, id)
	return c.Rdb.Del(ctx, key).Err()
}

func SetTaskSummaryCache(ctx context.Context, uid, pid int, status string, total int64, ts []TaskSummary) error {
	key := taskListKey(uid, pid, status)
	b, err := json.Marshal(TaskListCache{Items: ts,Total: total})
	if err != nil {
		return err
	}
	return c.Rdb.Set(ctx, key, b, time.Hour).Err()
}

func GetTaskSummaryCache(ctx context.Context, uid, pid int, status string) (*TaskListCache, error){
	key := taskListKey(uid, pid, status)
	data, err := c.Rdb.Get(ctx, key).Bytes()
	if err == nil {
		var td TaskListCache
		if uerr := json.Unmarshal(data, &td); uerr != nil {
			return nil, uerr
		}
		return  &td, nil
	}
	return nil, err
}

func DelTaskSummaryCache(ctx context.Context, uid, pid int, status string) error {
	key := taskListKey(uid, pid, status)
	return c.Rdb.Del(ctx, key).Err()
}

