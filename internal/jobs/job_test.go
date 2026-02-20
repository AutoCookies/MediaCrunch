package jobs

import "testing"

func TestValidateTransition(t *testing.T) {
	if err := ValidateTransition(StateQueued, StateRunning); err != nil {
		t.Fatalf("expected valid transition: %v", err)
	}
	if err := ValidateTransition(StateQueued, StateSuccess); err == nil {
		t.Fatal("expected invalid transition error")
	}
}
