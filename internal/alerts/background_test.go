package alerts

import (
	"testing"
	"time"
)

func TestParseEvalInterval(t *testing.T) {
	tests := []struct {
		raw    string
		want   time.Duration
		enable bool
	}{
		{"", 0, false},
		{"off", 0, false},
		{"0", 0, false},
		{"30s", 0, false},
		{"5m", 5 * time.Minute, true},
		{"1h", time.Hour, true},
	}
	for _, tc := range tests {
		got, ok := ParseEvalInterval(tc.raw)
		if ok != tc.enable || got != tc.want {
			t.Fatalf("ParseEvalInterval(%q) = (%v,%v) want (%v,%v)", tc.raw, got, ok, tc.want, tc.enable)
		}
	}
}

func TestStartBackgroundEvaluatorNoop(t *testing.T) {
	cancel := StartBackgroundEvaluator(nil, time.Minute)
	cancel()
	cancel = StartBackgroundEvaluator(nil, 0)
	cancel()
}
