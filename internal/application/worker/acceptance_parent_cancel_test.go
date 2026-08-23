package worker

import (
	"context"
	"testing"
	"time"
)

type cancellationObserver struct {
	started chan struct{}
	done    chan struct{}
}

func (h cancellationObserver) Handle(ctx context.Context, _ Job) error {
	close(h.started)
	<-ctx.Done()
	close(h.done)
	return ctx.Err()
}

func TestParentCancellationReachesActiveJob(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	handler := cancellationObserver{started: make(chan struct{}), done: make(chan struct{})}
	runner := &Runner{}
	finished := make(chan struct{})
	go func() {
		runner.executeJob(parent, Job{ID: "job-1"}, handler)
		close(finished)
	}()
	<-handler.started
	cancel()
	select {
	case <-handler.done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("active handler did not observe parent cancellation")
	}
	<-finished
}
