package deviceapp

import (
	"context"
	"testing"
	"time"

	"github.com/wyw14/cry-082/internal/domain/device"
	"github.com/wyw14/cry-082/internal/domain/site"
	clockpkg "github.com/wyw14/cry-082/internal/platform/clock"
	"github.com/wyw14/cry-082/internal/platform/idempotency"
	"github.com/wyw14/cry-082/internal/repository/memory"
)

func TestReplacementTransitionKeepsSuccessorLink(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	store := memory.New()
	current := device.Device{ID: "device-old", SiteID: "site-1", Status: device.StatusOnline, Version: 3}
	if err := store.Save(ctx, current); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveMembership(ctx, site.Membership{UserID: "maintainer-1", SiteID: "site-1", Role: site.RoleMaintainer}); err != nil {
		t.Fatal(err)
	}
	service := New(store, store, store, memory.TransactionManager{}, clockpkg.NewManual(now), &idempotency.Generator{})
	updated, err := service.Transition(ctx, TransitionInput{DeviceID: "device-old", Next: device.StatusReplaced, ReplacementID: "device-new", ActorID: "maintainer-1", Reason: "replace failed sensor", RequestID: "request-1", ExpectedVersion: 3})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != device.StatusReplaced || updated.ReplacementID != "device-new" {
		t.Fatalf("replacement transition lost successor: %+v", updated)
	}
}
