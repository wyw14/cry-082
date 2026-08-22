package alert

import (
	"errors"
	"testing"
	"time"
)

func TestEnvironmentalAndOfflineAlertsStaySeparate(t *testing.T) {
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	environmental, err := New("a1", "s1", "p1", "d1", KindEnvironmentalExceedance, "r1", 2, "s1:p1:r1:2", now)
	if err != nil {
		t.Fatal(err)
	}
	offline, err := New("a2", "s1", "p1", "d1", KindDeviceOffline, "", 0, "d1:offline", now)
	if err != nil {
		t.Fatal(err)
	}
	if environmental.MergeKey == offline.MergeKey || environmental.Kind == offline.Kind {
		t.Fatal("alert causes were merged")
	}
	if err := offline.Transition(StatusDispatched, "u1", "assign", "u2", now); !errors.Is(err, ErrInvalidAlertTransition) {
		t.Fatalf("expected acknowledge before dispatch, got %v", err)
	}
	if err := offline.Transition(StatusAcknowledged, "u1", "seen", "", now); err != nil {
		t.Fatal(err)
	}
	if err := offline.Transition(StatusDispatched, "u1", "assign", "u2", now); err != nil {
		t.Fatal(err)
	}
}
