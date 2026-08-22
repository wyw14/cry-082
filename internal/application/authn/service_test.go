package authn

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/wyw14/cry-082/internal/domain/auth"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

type testIssuer struct{ now time.Time }

func (i testIssuer) IssueAccessToken(ctx context.Context, user auth.User, ttl time.Duration) (string, time.Time, error) {
	return "access-" + user.ID, i.now.Add(ttl), nil
}

func TestRefreshRotationRevokesPreviousToken(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	clock := clockpkg.NewManual(now)
	store := memory.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("secure-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := auth.NewUser("u1", "operator", "操作员", hash)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	service := New(store, testIssuer{now: now}, clock, &idempotency.Generator{}, 15*time.Minute, 24*time.Hour)
	first, err := service.Login(ctx, "operator", "secure-password")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.SplitN(first.RefreshToken, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("refresh=%q", first.RefreshToken)
	}
	second, err := service.Refresh(ctx, parts[0], parts[1])
	if err != nil {
		t.Fatal(err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if _, err := service.Refresh(ctx, parts[0], parts[1]); err == nil {
		t.Fatal("revoked refresh token was accepted")
	}
}

func TestConcurrentRefreshOnlyOneRotationSucceeds(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	clock := clockpkg.NewManual(now)
	store := memory.New()
	hash, err := bcrypt.GenerateFromPassword([]byte("DustDemo!2026"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := auth.NewUser("u-race", "race.user", "Race User", hash)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveUser(ctx, user); err != nil {
		t.Fatal(err)
	}
	service := New(store, testIssuer{now: now}, clock, &idempotency.Generator{}, 15*time.Minute, 24*time.Hour)
	pair, err := service.Login(ctx, "race.user", "DustDemo!2026")
	if err != nil {
		t.Fatal(err)
	}
	tokenID, raw, ok := strings.Cut(pair.RefreshToken, ".")
	if !ok {
		t.Fatal("refresh token has no separator")
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			ready.Done()
			<-start
			_, err := service.Refresh(ctx, tokenID, raw)
			results <- err
		}()
	}
	ready.Wait()
	close(start)
	var success, rejected int
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, auth.ErrInvalidToken) {
			rejected++
		} else {
			t.Fatalf("unexpected refresh result: %v", err)
		}
	}
	if success != 1 || rejected != 1 {
		t.Fatalf("success=%d rejected=%d", success, rejected)
	}
}
