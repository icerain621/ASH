package memory

import (
	"testing"
	"time"
)

func TestParseSweepInterval(t *testing.T) {
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
		{"24h", 24 * time.Hour, true},
	}
	for _, tc := range tests {
		got, ok := ParseSweepInterval(tc.raw)
		if ok != tc.enable || got != tc.want {
			t.Fatalf("ParseSweepInterval(%q) = (%v,%v) want (%v,%v)", tc.raw, got, ok, tc.want, tc.enable)
		}
	}
}

func TestStartBackgroundTTLSweepNoop(t *testing.T) {
	cancel := StartBackgroundTTLSweep(nil, time.Hour)
	cancel()
	cancel = StartBackgroundTTLSweep(nil, 0)
	cancel()
}
