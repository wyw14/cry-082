package rule

import (
	"testing"
	"time"
)

func TestRequestedRecalculationRemainsPendingForWorker(t *testing.T) {
	now := time.Date(2026, 8, 23, 8, 0, 0, 0, time.UTC)
	plan := RecalculationPlan{SiteID: "site", RuleID: "rule", Reason: "verify revised limits", ActorID: "supervisor", FromVersion: 1, ToVersion: 2, WindowStart: now.Add(-time.Hour), WindowEnd: now}
	job, err := plan.Schedule("recalculation", now)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != RecalculationPending {
		t.Fatalf("new recalculation cannot be claimed by worker: status=%s", job.Status)
	}
}
