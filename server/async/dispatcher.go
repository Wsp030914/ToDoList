package async

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

var lg *zap.Logger

type Job struct {
	Type    string
	Payload []byte
	TraceID string
	Retry   int
}

type TimeoutPolicy struct {
	JobTimeout     time.Duration
	AttemptTimeout time.Duration
	MaxRetry       int
}

type Handler func(ctx context.Context, job Job, lg *zap.Logger) error
type Dispatcher struct {
	handlers map[string]Handler
	jobs     chan Job
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	Policy   map[string]TimeoutPolicy
}

func NewDispatcher(buf int) *Dispatcher {
	lg = zap.L()
	ctx, cancel := context.WithCancel(context.Background())
	return &Dispatcher{
		handlers: make(map[string]Handler),
		jobs:     make(chan Job, buf),
		ctx:      ctx,
		cancel:   cancel,
		Policy:   make(map[string]TimeoutPolicy),
	}
}

func (d *Dispatcher) Start(workers int) {
	for i := 0; i < workers; i++ {
		d.wg.Add(1)
		go d.worker(i)
	}
}

func (d *Dispatcher) Stop() {
	d.cancel()
	d.wg.Wait()
}

func (d *Dispatcher) Enqueue(j Job) bool {
	select {
	case <-d.ctx.Done():
		lg.Error("[Dispatcher] stopped, reject job:" + j.Type + j.TraceID)
		return false
	default:
	}

	select {
	case d.jobs <- j:
		return true
	default:
		lg.Error("[Dispatcher] job queue full, drop job:" + j.Type + j.TraceID)
		return false
	}
}

func (d *Dispatcher) Register(jobType string, h Handler, policy TimeoutPolicy) {
	if _, exists := d.handlers[jobType]; exists {
		panic("duplicate job handler: " + jobType)
	}
	d.handlers[jobType] = h
	if _, exists := d.Policy[jobType]; exists {
		panic("duplicate job Policy: " + jobType)
	}
	d.Policy[jobType] = policy
}

func (d *Dispatcher) worker(id int) {
	defer d.wg.Done()

	for {
		select {
		case <-d.ctx.Done():
			lg.Error("[Worker]" + strconv.Itoa(id) + "exit")
			return
		case j, ok := <-d.jobs:
			if !ok {
				lg.Info("[Worker]" + strconv.Itoa(id) + " jobs closed, exit")
				return
			}
			ctx, cancel := context.WithTimeout(d.ctx, d.Policy[j.Type].JobTimeout)
			err := d.safeHandle(ctx, j, id)
			cancel()
			if err != nil {
				lg.Error("[Worker]" + strconv.Itoa(id) + "handle failed:" + j.Type + j.TraceID)
			}
		}
	}
}

func (d *Dispatcher) safeHandle(ctx context.Context, job Job, worked int) (err error) {
	defer func() {
		if r := recover(); r != nil {

			lg.Error("handler panic",
				zap.Any("panic", r),
				zap.ByteString("stack", debug.Stack()),
				zap.String("job_type", job.Type),
				zap.String("request_id", job.TraceID),
				zap.Int("worker_id", worked),
			)

			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	err = d.handle(ctx, job, worked)
	return
}

func (d *Dispatcher) handle(ctx context.Context, job Job, worked int) error {
	h, ok := d.handlers[job.Type]
	if !ok {
		return errors.New("no handler for job type")
	}
	hlg := lg.With(
		zap.String("job_type", job.Type),
		zap.String("request_id", job.TraceID),
		zap.Int("worker_id", worked),
	)
	attemptLg := hlg.With(zap.Int("retry", job.Retry))
	attemptCtx, cancel := context.WithTimeout(ctx, d.Policy[job.Type].AttemptTimeout)
	err := h(attemptCtx, job, attemptLg)
	cancel()

	for err != nil && job.Retry < d.Policy[job.Type].MaxRetry {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, context.Canceled) {
			return err
		}

		backoff := 200 * time.Millisecond
		t := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
			job.Retry = job.Retry + 1
			attemptLg = hlg.With(zap.Int("retry", job.Retry))
			attemptCtx, cancel = context.WithTimeout(ctx, d.Policy[job.Type].AttemptTimeout)
			err = h(attemptCtx, job, attemptLg)
			cancel()
		}

	}
	if err != nil {
		lg.Error(job.Type+"exceed fail", zap.Int("retry", job.Retry), zap.Error(err))
		return err
	}
	return nil
}
