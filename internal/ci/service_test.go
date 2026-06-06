package ci

import (
	"context"
	"testing"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestDiagnoseLogClassifiesTestFailure(t *testing.T) {
	resp := DiagnoseLog("go test ./...\n--- FAIL: TestThing (0.01s)\nFAIL\tgithub.com/acme/app\t0.1s\n")
	if resp.RootCause != "test_failure" {
		t.Fatalf("rootCause=%q want test_failure", resp.RootCause)
	}
	if resp.Confidence <= 0.8 || len(resp.EvidenceRefs) == 0 || resp.LogDigest == "" {
		t.Fatalf("resp=%+v want confident evidence with digest", resp)
	}
}

func TestServiceDiagnosePersistsLogText(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := NewService(db, nil)
	resp, err := svc.Diagnose(context.Background(), DiagnoseRequest{
		SpaceID: "local",
		LogText: "go test ./...\nundefined: MissingType\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RootCause != "go_compile_failure" {
		t.Fatalf("rootCause=%q want go_compile_failure", resp.RootCause)
	}
	var rows []store.CIDiagnosis
	if err := db.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].RootCause != resp.RootCause || rows[0].LogDigest == "" {
		t.Fatalf("rows=%+v want persisted diagnosis", rows)
	}
}
