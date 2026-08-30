package audit

import (
	"testing"
	"time"
)

func sampleRecord() Record {
	th := "deadbeef"
	cv := "connector-mock-network-block@1.0.0"
	caseID := "case-001"
	return Record{
		TenantID:         "tenant-lab",
		Actor:            "analyst",
		ActorType:        "analyst",
		CaseID:           &caseID,
		Action:           "case.transition",
		TargetHash:       &th,
		PolicyVersion:    "policy-1.0",
		ConnectorVersion: &cv,
		Decision:         "completed",
		Timestamp:        time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC),
	}
}

func TestResultHash_Deterministic(t *testing.T) {
	r1 := sampleRecord()
	r2 := sampleRecord()
	if resultHash(r1) != resultHash(r2) {
		t.Errorf("resultHash is not deterministic for identical records")
	}
}

func TestResultHash_DetectsTamper(t *testing.T) {
	r := sampleRecord()
	original := resultHash(r)

	tampered := r
	tampered.Decision = "denied"
	if resultHash(tampered) == original {
		t.Errorf("resultHash did not change when decision was tampered")
	}

	tampered2 := r
	otherHash := "cafebabe"
	tampered2.TargetHash = &otherHash
	if resultHash(tampered2) == original {
		t.Errorf("resultHash did not change when target_hash was tampered")
	}
}

func TestTargetHash_HidesRawValue(t *testing.T) {
	raw := "198.51.100.10"
	hashed := TargetHash(raw)
	if hashed == raw {
		t.Fatalf("TargetHash returned the raw value unchanged")
	}
	if len(hashed) != 64 {
		t.Errorf("TargetHash() length = %d, want 64 (hex sha256)", len(hashed))
	}
	if TargetHash(raw) != hashed {
		t.Errorf("TargetHash is not deterministic")
	}
}
