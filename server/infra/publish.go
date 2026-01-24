package infra

import (
	"ToDoList/server/async"
	"context"
	"go.uber.org/zap"
	"time"
)

func Publish(bus *async.EventBus, lg *zap.Logger, topic string, payload any, timeout time.Duration, fields ...zap.Field) bool {

	pubCtx, cancel := context.WithTimeout(context.Background(), timeout)
	ok := bus.Publish(pubCtx, topic, payload)
	cancel()

	if !ok {
		lg.Warn("bus.publish_failed",
			append([]zap.Field{zap.String("topic", topic)}, fields...)...,
		)
	}
	return ok
}
