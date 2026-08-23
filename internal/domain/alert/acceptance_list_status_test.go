package alert

import "testing"

func TestLaterPageKeepsAlertStatusFilter(t *testing.T) {
	scope, err := NewListScope("site-1", KindEnvironmentalExceedance, StatusAcknowledged, 20, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got := scope.EffectiveStatus(); got != StatusAcknowledged {
		t.Fatalf("later page status=%q, want %q", got, StatusAcknowledged)
	}
}
