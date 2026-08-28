package goal

import "testing"

func TestPickScenario(t *testing.T) {
	cases := []struct {
		goal string
		want string
	}{
		{"fix CVE-2024-1 in auth", "security_patch"},
		{"紧急热修线上支付", "hotfix"},
		{"Add dark mode to settings", "feature_delivery"},
		{"security_patch needed", "security_patch"},
		{"hotfix prod outage", "hotfix"},
	}
	for _, tc := range cases {
		got, _ := pickScenario(tc.goal)
		if got != tc.want {
			t.Fatalf("goal=%q got=%q want=%q", tc.goal, got, tc.want)
		}
	}
}
