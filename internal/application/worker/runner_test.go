package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
)

type testQueue struct {
	mu          sync.Mutex
	job         *Job
	acked, dead bool
}

func (q *testQueue) Claim(ctx context.Context, now time.Time) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.job == nil || q.acked || q.dead || q.job.AvailableAt.After(now) {
		return nil, nil
	}
	copy := *q.job
	return &copy, nil
}
func (q *testQueue) Ack(ctx context.Context, id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.acked = true
	return nil
}
func (q *testQueue) Retry(ctx context.Context, job Job, next time.Time, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	job.AvailableAt = next
	q.job = &job
	return nil
}
func (q *testQueue) DeadLetter(ctx context.Context, job Job, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.dead = true
	return nil
}

type failingHandler struct{}

func (failingHandler) Handle(context.Context, Job) error { return errors.New("offline adapter") }
func TestRunnerHonorsCancellation(t *testing.T) {
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	clock := clockpkg.NewManual(now)
	queue := &testQueue{job: &Job{ID: "j1", Kind: "notify", AvailableAt: now}}
	runner := New(queue, map[string]Handler{"notify": failingHandler{}}, clock, 2, 0, time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		queue.mu.Lock()
		dead := queue.dead
		queue.mu.Unlock()
		if dead {
			break
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runner did not stop")
	}
	queue.mu.Lock()
	defer queue.mu.Unlock()
	if !queue.dead {
		t.Fatal("job was not dead-lettered after retry limit")
	}
}
