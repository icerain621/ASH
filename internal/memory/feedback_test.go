package memory

import "testing"

func TestRecordFeedbackEmitsThreeCounters(t *testing.T) {
	svc, ev := newTestMemory(t)
	seedRun(t, svc.db, "run_fb", "trace_fb")
	for _, kind := range []string{FeedbackHitUsed, FeedbackUsedCorrectly, FeedbackWrongUpdate} {
		if err := svc.RecordFeedback(FeedbackRequest{
			RunID: "run_fb", RecordID: "mem_1", Kind: kind,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.RecordFeedback(FeedbackRequest{RecordID: "mem_1", Kind: "promote"}); err == nil {
		t.Fatal("expected unknown kind to fail")
	}
	events, err := ev.ListAfter("run_fb", 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, event := range events {
		if event.Type != "memory.feedback" {
			continue
		}
		kinds = append(kinds, kindFromPayload(string(event.Payload)))
	}
	got := TallyFeedback(kinds)
	if got.HitUsed != 1 || got.UsedCorrectly != 1 || got.WrongUpdate != 1 {
		t.Fatalf("counts=%+v kinds=%v", got, kinds)
	}
}

func kindFromPayload(payload string) string {
	const key = `"kind":"`
	i := indexOf(payload, key)
	if i < 0 {
		return ""
	}
	rest := payload[i+len(key):]
	j := indexOf(rest, `"`)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
