package runs

import (
	"errors"
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/store"
)

func TestEnforceSpaceQuotasConcurrent(t *testing.T) {
	svc, _ := testRunsService(t)
	space := "sp_quota"
	now := time.Now().UTC()
	if err := svc.gdb().Create(&store.SpacePolicyPack{
		SpaceID: space, CitationMode: "optional", BodyJSON: `{"quotas":{"maxConcurrentRuns":1}}`,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.gdb().Create(&store.RunRecord{
		ID: space + "_run1", TraceID: "trc_q1", ScenarioName: "feature_delivery", ScenarioVersion: "1.0.0",
		PolicyProfile: "default", Status: StatusRunning, SpaceID: space, StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	err := svc.enforceSpaceQuotas(space)
	if !errors.Is(err, ErrSpaceQuotaExceeded) {
		t.Fatalf("want ErrSpaceQuotaExceeded got %v", err)
	}
	// Finished run does not count.
	_ = svc.gdb().Model(&store.RunRecord{}).Where("id = ?", space+"_run1").Update("status", StatusFinished).Error
	if err := svc.enforceSpaceQuotas(space); err != nil {
		t.Fatalf("after finish: %v", err)
	}
	// Unlimited when no pack / no quotas.
	if err := svc.enforceSpaceQuotas("local"); err != nil {
		t.Fatal(err)
	}
}
