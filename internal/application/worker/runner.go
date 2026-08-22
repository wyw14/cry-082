package worker

import (
	"context"
	"errors"
	"math"
	"time"
)

var ErrDeadLettered = errors.New("job moved to dead letter")

type Job struct {
	ID, Kind    string
	Payload     []byte
	Attempt     int
	AvailableAt time.Time
}
type Queue interface {
	Claim(context.Context, time.Time) (*Job, error)
	Ack(context.Context, string) error
	Retry(context.Context, Job, time.Time, string) error
	DeadLetter(context.Context, Job, string) error
}
type Handler interface {
	Handle(context.Context, Job) error
}
type Clock interface{ Now() time.Time }

type Runner struct {
	queue                       Queue
	handlers                    map[string]Handler
	clock                       Clock
	maximumAttempts             int
	baseBackoff, maximumBackoff time.Duration
}

func New(queue Queue, handlers map[string]Handler, clock Clock, maximumAttempts int, baseBackoff, maximumBackoff time.Duration) *Runner {
	return &Runner{queue: queue, handlers: handlers, clock: clock, maximumAttempts: maximumAttempts, baseBackoff: baseBackoff, maximumBackoff: maximumBackoff}
}

func (r *Runner) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		job, err := r.queue.Claim(ctx, r.clock.Now())
		if err != nil {
			return err
		}
		if job == nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				continue
			}
		}
		handler, ok := r.handlers[job.Kind]
		if !ok {
			if err := r.queue.DeadLetter(ctx, *job, "unknown job kind"); err != nil {
				return err
			}
			continue
		}
		attemptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		handleErr := handler.Handle(attemptCtx, *job)
		cancel()
		if handleErr == nil {
			if err := r.queue.Ack(ctx, job.ID); err != nil {
				return err
			}
			continue
		}
		job.Attempt++
		if job.Attempt >= r.maximumAttempts {
			if err := r.queue.DeadLetter(ctx, *job, handleErr.Error()); err != nil {
				return err
			}
			continue
		}
		backoff := time.Duration(float64(r.baseBackoff) * math.Pow(2, float64(job.Attempt-1)))
		if backoff > r.maximumBackoff {
			backoff = r.maximumBackoff
		}
		if err := r.queue.Retry(ctx, *job, r.clock.Now().Add(backoff), handleErr.Error()); err != nil {
			return err
		}
	}
}
