package worker

import (
	"context"
	"math"
	"time"
)

type executionStatus string

const (
	executionRunning   executionStatus = "running"
	executionSucceeded executionStatus = "succeeded"
	executionRetry     executionStatus = "retry"
	executionDead      executionStatus = "dead-lettered"
)

type jobExecution struct {
	job       Job
	status    executionStatus
	handleErr error
}

func (r *Runner) executeJob(ctx context.Context, job Job, handler Handler) jobExecution {
	attemptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	execution := jobExecution{job: job, status: executionRunning}
	execution.handleErr = handler.Handle(attemptCtx, job)
	if execution.handleErr == nil {
		execution.status = executionSucceeded
		return execution
	}
	execution.job.Attempt++
	if execution.job.Attempt >= r.maximumAttempts {
		execution.status = executionDead
		return execution
	}
	execution.status = executionRetry
	return execution
}

func (r *Runner) settle(ctx context.Context, execution jobExecution) error {
	switch execution.status {
	case executionSucceeded:
		return r.queue.Ack(ctx, execution.job.ID)
	case executionDead:
		return r.queue.DeadLetter(ctx, execution.job, execution.handleErr.Error())
	case executionRetry:
		backoff := time.Duration(float64(r.baseBackoff) * math.Pow(2, float64(execution.job.Attempt-1)))
		if backoff > r.maximumBackoff {
			backoff = r.maximumBackoff
		}
		return r.queue.Retry(ctx, execution.job, r.clock.Now().Add(backoff), execution.handleErr.Error())
	default:
		return nil
	}
}
