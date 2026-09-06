package config

import "testing"

func TestRegionDefaultAndOverride(t *testing.T) {
	t.Setenv("ASH_REGION", "")
	if got := Region(); got != "default" {
		t.Fatalf("Region()=%q want default", got)
	}
	t.Setenv("ASH_REGION", "  ap-east-1  ")
	if got := Region(); got != "ap-east-1" {
		t.Fatalf("Region()=%q want ap-east-1", got)
	}
}
