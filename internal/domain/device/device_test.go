package device

import (
	"errors"
	"testing"
	"time"
)

func TestDeviceLifecycle(t *testing.T) {
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	value, err := New("d1", "D-001", "Sim", "s1", "p1", "gate", NetworkConfig{Host: "sim.local", Port: 1883, Protocol: "mqtt"})
	if err != nil {
		t.Fatal(err)
	}
	if err := value.MarkSeen(now); err != nil {
		t.Fatal(err)
	}
	if value.Status != StatusOnline {
		t.Fatalf("status=%s", value.Status)
	}
	if err := value.Transition(StatusMaintenance, now.Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	if err := value.Transition(StatusReplaced, now.Add(2*time.Hour), ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected transition error, got %v", err)
	}
	if err := value.Transition(StatusReplaced, now.Add(2*time.Hour), "d2"); err != nil {
		t.Fatal(err)
	}
}
