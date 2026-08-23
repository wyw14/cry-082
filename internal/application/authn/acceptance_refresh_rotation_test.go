package authn

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/auth"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

type synchronizedRefreshRepository struct {
	*memory.Store
	arrived chan struct{}
	release chan struct{}
}

func (r *synchronizedRefreshRepository) FindRefreshToken(ctx context.Context, id string) (auth.RefreshToken, error) {
	token, err := r.Store.FindRefreshToken(ctx, id)
	r.arrived <- struct{}{}
	select {
	case <-r.release:
		return token, err
	case <-ctx.Done():
		return auth.RefreshToken{}, ctx.Err()
	}
}

func TestConcurrentRefreshRotationHasSingleWinner(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	user, err := auth.NewUser("user-1", "operator", "Operator", []byte("password-hash"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	raw := "0123456789abcdef0123456789abcdef"
	token, err := auth.NewRefreshToken("refresh-1", user.ID, raw, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRefreshToken(ctx, token); err != nil {
		t.Fatal(err)
	}
	repository := &synchronizedRefreshRepository{Store: store, arrived: make(chan struct{}, 2), release: make(chan struct{})}
	service := New(repository, testIssuer{now: now}, clockpkg.NewManual(now), &idempotency.Generator{}, 15*time.Minute, time.Hour)

	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for index := 0; index < 2; index++ {
		go func() {
			ready.Done()
			_, err := service.Refresh(ctx, token.ID, raw)
			results <- err
		}()
	}
	ready.Wait()
	<-repository.arrived
	<-repository.arrived
	close(repository.release)

	success, rejected := 0, 0
	for index := 0; index < 2; index++ {
		err := <-results
		switch {
		case err == nil:
			success++
		case errors.Is(err, auth.ErrInvalidToken):
			rejected++
		default:
			t.Fatalf("unexpected refresh error: %v", err)
		}
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("success=%d rejected=%d", success, rejected)
	}
}
