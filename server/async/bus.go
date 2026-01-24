package async

import (
	"ToDoList/server/reqctx"
	"context"
	"encoding/json"
)

type EventBus struct {
	d *Dispatcher
}

func NewEventBus(d *Dispatcher) *EventBus {
	return &EventBus{d: d}
}

func (b *EventBus) Publish(ctx context.Context, jobType string, payload any) bool {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return false
		default:
		}
	}
	lg := reqctx.LoggerFromContext(ctx)
	reqID := reqctx.RequestIDFromCtx(ctx)
	bs, err := json.Marshal(payload)
	if err != nil {
		lg.Warn("async.Publish.payload_Marshal_error")
		return false
	}
	j := Job{
		Type:    jobType,
		Payload: bs,
		TraceID: reqID,
	}
	return b.d.Enqueue(j)
}
