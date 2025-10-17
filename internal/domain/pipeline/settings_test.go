package pipeline

import "testing"

func TestSettingsApplyDefaults(t *testing.T) {
	s := Settings{}
	defaulted := s.ApplyDefaults()
	if defaulted.Parallel != 4 {
		t.Fatalf("expected parallel default 4, got %d", defaulted.Parallel)
	}
	if defaulted.Timeout != 300 {
		t.Fatalf("expected timeout default 300, got %d", defaulted.Timeout)
	}
}
