package signatures

import "testing"

func TestMatchEventClass_SSHBruteforce(t *testing.T) {
	sig := MatchEventClass("authentication")
	if sig == nil {
		t.Fatal("expected a signature for authentication events")
	}
	if sig.ID != "ssh-bruteforce" || sig.ObservableType != "ipv4" || sig.Threshold != 5 || sig.WindowSeconds != 60 {
		t.Fatalf("unexpected signature: %+v", sig)
	}
}

func TestMatchEventClass_NoMatch(t *testing.T) {
	if sig := MatchEventClass("network"); sig != nil {
		t.Fatalf("expected no signature for network events, got %+v", sig)
	}
}
