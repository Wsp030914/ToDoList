package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"hash/fnv"
	"strings"
	"time"
)

type ProjectListCache struct {
	Items []ProjectSummary `json:"items"`
	Total int64            `json:"total"`
}

func projectsVerKey(uid int) string {
	return fmt.Sprintf("u:%d:projects:ver", uid)
}
func projectsNoCacheKey(uid int) string {
	return fmt.Sprintf("u:%d:projects:nocache", uid)
}
func projectsListKey(uid int, ver int64, name string, page, size int) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "-"
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return fmt.Sprintf("u:%d:projects:list:v%d:n%016x:p%d:s%d", uid, ver, h.Sum64(), page, size)
}

func GetProjectsSummaryCache(ctx context.Context, uid int, name string, page, size int, ver int64) ([]ProjectSummary, int64, error) {
	key := projectsListKey(uid, ver, name, page, size)
	b, err := c.Rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, 0, err 
	}

	var cached ProjectListCache
	if err := json.Unmarshal(b, &cached); err != nil {
		_ = c.Rdb.Del(context.Background(), key).Err()
		return nil, 0, redis.Nil
	}
	return cached.Items, cached.Total, nil
}

func PutProjectsSummaryCache(ctx context.Context, uid int, name string, page, size int, total int64, ps []ProjectSummary, ver int64) error {
	key := projectsListKey(uid, ver, name, page, size)
	val, err := json.Marshal(ProjectListCache{Items: ps, Total: total})
	if err != nil {
		return err
	}
	return c.Rdb.Set(ctx, key, val, 30*time.Second).Err()
}

func GetProjectsVer(ctx context.Context, uID int) int64 {
	key := fmt.Sprintf("u:%d:projects:ver", uID)
	v, err := c.Rdb.Get(ctx, key).Int64()
	if errors.Is(err, redis.Nil) || err != nil || v < 1 {
		_ = c.Rdb.SetNX(ctx, key, 1, 0).Err()
		return 1
	}
	return v
}

func ShouldBypassProjectsCache(ctx context.Context, uid int) bool {
	n, err := c.Rdb.Exists(ctx, projectsNoCacheKey(uid)).Result()
	return err == nil && n == 1
}

func IncrProjectsVer(ctx context.Context, rdb *redis.Client, uid int) error {
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	if err := rdb.Incr(ctx, projectsVerKey(uid)).Err(); err != nil {
		_ = rdb.Set(ctx, projectsNoCacheKey(uid), 1, 5*time.Second).Err()
		return err
	}
	return nil
}
