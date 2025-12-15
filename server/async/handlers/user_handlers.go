package handlers

import (
	"ToDoList/server/async"
	"ToDoList/server/service"
	"ToDoList/server/utils"
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"time"
)

type cosDeletePayload struct {
	Key string `json:"key"`
}
type avatarKeyPut struct {
	UID       int    `json:"uid"`
	AvatarKey string `json:"avatarKey"`
}

type avatarKeyDel struct {
	UID int `json:"uid"`
}

type putVersion struct {
	UID          int `json:"uid"`
	TokenVersion int `json:"tokenVersion"`
}

func DeleteCosObject(ctx context.Context, job async.Job, lg *zap.Logger) error {
	var p cosDeletePayload
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		lg.Error(job.Type + job.TraceID + "Payload Unmarshal is err")
		return nil
	}
	if p.Key == "" {
		lg.Error(job.Type + job.TraceID + "cosKey is nil")
		return nil
	}
	rctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	return utils.DeleteObject(rctx, p.Key)
}

func UpdateAvatarKey(ctx context.Context, job async.Job, lg *zap.Logger) error {
	var a avatarKeyPut
	if err := json.Unmarshal(job.Payload, &a); err != nil {
		lg.Error(job.Type + job.TraceID + "Payload Unmarshal is err")
		return nil
	}
	if a.UID <= 0 || a.AvatarKey == "" {
		lg.Error(job.Type + job.TraceID + "avatarKey is nil or UID <= 0")
		return nil
	}

	rctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	return service.UpdateAvatarKey(rctx, a.UID, a.AvatarKey)
}

func PutVersion(ctx context.Context, job async.Job, lg *zap.Logger) error {
	var p putVersion
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		lg.Error(job.Type+"Payload Unmarshal is err", zap.Error(err))
		return nil
	}
	if p.UID <= 0 {
		lg.Error(job.Type + job.TraceID + "UID <= 0")
		return nil
	}
	if p.TokenVersion <= 0 {
		lg.Error(job.Type + job.TraceID + "TokenVersion <= 0")
		return nil
	}
	return service.PutVersion(ctx, p.UID, p.TokenVersion)
}
