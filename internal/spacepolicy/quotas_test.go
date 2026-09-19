package spacepolicy

import "testing"

func TestQuotasFromBodyJSON(t *testing.T) {
	q, err := QuotasFromBodyJSON(`{"quotas":{"maxConcurrentRuns":2,"tokenBudgetProxy":100}}`)
	if err != nil {
		t.Fatal(err)
	}
	if q.MaxConcurrentRuns != 2 || q.TokenBudgetProxy != 100 {
		t.Fatalf("got %+v", q)
	}
	q, err = QuotasFromBodyJSON(`{}`)
	if err != nil || q.MaxConcurrentRuns != 0 {
		t.Fatalf("empty=%+v err=%v", q, err)
	}
	if _, err := QuotasFromBodyJSON(`{"quotas":{"maxConcurrentRuns":-1}}`); err == nil {
		t.Fatal("expected negative max error")
	}
}

func TestStricterQuotas(t *testing.T) {
	got := StricterQuotas(Quotas{MaxConcurrentRuns: 5}, Quotas{MaxConcurrentRuns: 2})
	if got.MaxConcurrentRuns != 2 {
		t.Fatalf("got %d", got.MaxConcurrentRuns)
	}
	got = StricterQuotas(Quotas{}, Quotas{MaxConcurrentRuns: 3})
	if got.MaxConcurrentRuns != 3 {
		t.Fatalf("unlimited vs 3 => %d", got.MaxConcurrentRuns)
	}
}
