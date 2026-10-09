package review

import "testing"

func TestSubmitAndDecide(t *testing.T) {
	svc := NewService()
	item, err := svc.Submit(KindMemoryRecord, "mem_1", "consolidate")
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != StatusPending {
		t.Fatalf("%+v", item)
	}
	if len(svc.List(KindMemoryRecord)) != 1 {
		t.Fatal("want one pending")
	}
	decided, err := svc.Decide(item.ID, "approve", "ok")
	if err != nil {
		t.Fatal(err)
	}
	if decided.Status != StatusApproved {
		t.Fatalf("%+v", decided)
	}
	if len(svc.List("")) != 0 {
		t.Fatal("queue should be empty")
	}
}

func TestRejectUnknownKind(t *testing.T) {
	svc := NewService()
	if _, err := svc.Submit(Kind("nope"), "x", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestMemoryEvalCounts(t *testing.T) {
	got := MemoryEvalCounts([]string{"hit_used", "hit_used", "used_correctly", "wrong_update"})
	if got.Remembered != 2 || got.UsedCorrectly != 1 || got.WrongUpdate != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestQualityCaseIDsNonEmpty(t *testing.T) {
	if len(QualityCaseIDs()) == 0 {
		t.Fatal("expected quality case ids")
	}
}
