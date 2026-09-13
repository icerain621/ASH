package scoring_test

import (
	"testing"

	"github.com/ash-repwiki/ash/internal/scoring"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestRubricValidateAndComposite(t *testing.T) {
	r := scoring.ReviewRubric{Correctness: 4, Safety: 5, Citable: 3, Efficiency: 4}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := r.Composite(); got != 4.0 {
		t.Fatalf("composite=%v want 4", got)
	}
	bad := scoring.ReviewRubric{Correctness: 0, Safety: 5, Citable: 3, Efficiency: 4}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected validate error")
	}
}

func TestRecordScore(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := scoring.NewService(db)
	r := scoring.ReviewRubric{Correctness: 2, Safety: 2, Citable: 1, Efficiency: 2}
	ev, err := svc.RecordScore("local", "memory", "mem_1", "run_1", r, "tester", "weak")
	if err != nil {
		t.Fatal(err)
	}
	if ev.Composite != r.Composite() {
		t.Fatalf("composite=%v", ev.Composite)
	}
	var count int64
	if err := db.Model(&store.ScoreEvent{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
}

func TestDefaultRubricSchema(t *testing.T) {
	s := scoring.DefaultRubricSchema()
	if s.ID == "" || len(s.Dimensions) != 4 {
		t.Fatalf("%+v", s)
	}
}
