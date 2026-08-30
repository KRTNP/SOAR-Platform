package enrichment

import "testing"

func TestVerdictConfidence(t *testing.T) {
	tests := []struct {
		verdict string
		want    float64
	}{
		{"malicious", 0.9},
		{"suspicious", 0.6},
		{"benign", 0.8},
		{"unknown", 0.0},
		{"", 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.verdict, func(t *testing.T) {
			if got := verdictConfidence(tt.verdict); got != tt.want {
				t.Errorf("verdictConfidence(%q) = %v, want %v", tt.verdict, got, tt.want)
			}
		})
	}
}
