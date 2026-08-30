package runs

import (
	"strings"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/rules"
	"github.com/ash-repwiki/ash/internal/store"
)

type stubImprove struct {
	calls int
	last  string
}

func (s *stubImprove) DraftFromVerifyFailure(spaceID, runID, stepID, detail string) (string, error) {
	s.calls++
	s.last = stepID + ":" + detail
	return "imp_test", nil
}

func TestVerifyHelpers(t *testing.T) {
	step := rules.Step{Retry: &rules.RetrySpec{MaxAttempts: 3, BackoffMs: 5}}
	if verifyMaxAttempts(step) != 3 {
		t.Fatal("max attempts")
	}
	if verifyBackoff(step) != 5*time.Millisecond {
		t.Fatal("backoff")
	}
	if verifyMaxAttempts(rules.Step{}) != 1 {
		t.Fatal("default attempts")
	}
}

func TestMaybeDraftImprove(t *testing.T) {
	stub := &stubImprove{}
	svc := &Service{improve: stub}
	rec := &store.RunRecord{ID: "run_1", TraceID: "trc_1", SpaceID: "local"}
	svc.maybeDraftImprove(rec, "run_1", "qa.verify", "test.run failed")
	if stub.calls != 1 || !strings.Contains(stub.last, "qa.verify") {
		t.Fatalf("%+v", stub)
	}
}
