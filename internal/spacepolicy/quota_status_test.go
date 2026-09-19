package spacepolicy

import "testing"

func TestBuildQuotaStatus(t *testing.T) {
	st, err := BuildQuotaStatus("sp1", `{"quotas":{"maxConcurrentRuns":3,"tokenBudgetProxy":50}}`, 2)
	if err != nil {
		t.Fatal(err)
	}
	if st.SpaceID != "sp1" || st.Limits.MaxConcurrentRuns != 3 || st.Limits.TokenBudgetProxy != 50 {
		t.Fatalf("%+v", st)
	}
	if st.Usage.ActiveConcurrentRuns != 2 || st.Usage.TokenBudgetProxyUsed != 0 {
		t.Fatalf("usage=%+v", st.Usage)
	}
	st, err = BuildQuotaStatus("local", `{}`, 0)
	if err != nil || st.Limits.MaxConcurrentRuns != 0 {
		t.Fatalf("unlimited: %+v err=%v", st, err)
	}
}
