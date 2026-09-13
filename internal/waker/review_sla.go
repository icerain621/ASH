package waker

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ash-repwiki/ash/internal/spacepolicy"
	"github.com/ash-repwiki/ash/internal/store"
)

const KindReviewSLA = "review_sla"

func (s *Service) runReviewSLA(duty store.WakerDuty, dryRun bool) (SweepResponse, error) {
	out := SweepResponse{OK: true, DryRun: dryRun, Action: "report"}
	if s.q() == nil {
		return out, fmt.Errorf("database unavailable")
	}
	spaceID := normalizeSpaceID(duty.SpaceID)
	hours := 72
	pol := spacepolicy.NewService(s.db)
	if eff, err := pol.EffectivePolicy(spaceID); err == nil && eff != nil && eff.ReviewSLAHours > 0 {
		hours = eff.ReviewSLAHours
	}
	cutoff := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)

	var tokens []string

	var mems []store.MemoryRecord
	if err := s.q().Where("space_id = ? AND status = ? AND created_at < ?", spaceID, "candidate", cutoff).
		Find(&mems).Error; err != nil {
		return out, err
	}
	for _, m := range mems {
		out.Matched++
		out.Flagged++
		tokens = append(tokens, "sla_breach:memory:"+m.ID)
	}

	var harnessRows []store.HarnessProfileVersion
	if err := s.q().Where("space_id = ? AND status IN ? AND updated_at < ?",
		spaceID, []string{"in_review", "pending_second"}, cutoff).
		Find(&harnessRows).Error; err != nil {
		return out, err
	}
	for _, h := range harnessRows {
		out.Matched++
		out.Flagged++
		tokens = append(tokens, "sla_breach:harness_profile:"+h.ID)
	}

	var patches []store.ScenarioPatchDraft
	if err := s.q().Where("space_id = ? AND status = ? AND updated_at < ?",
		spaceID, "in_review", cutoff).
		Find(&patches).Error; err != nil {
		return out, err
	}
	for _, p := range patches {
		out.Matched++
		out.Flagged++
		tokens = append(tokens, "sla_breach:scenario_patch:"+p.ID)
	}

	out.Summary = fmt.Sprintf("review_sla breaches=%d hours=%d", out.Flagged, hours)
	if len(tokens) > 0 {
		out.Summary += " " + strings.Join(tokens, " ")
	}

	if !dryRun && out.Flagged > 0 {
		sample := tokens
		if len(sample) > 8 {
			sample = sample[:8]
		}
		payload := fmt.Sprintf(`{"count":%d,"hours":%d,"sample":[%s]}`,
			out.Flagged, hours, quoteJoin(sample))
		_ = s.q().Create(&store.AuditLog{
			ID: "aud_" + uuid.NewString(), SpaceID: spaceID,
			ActorID: "waker", EventType: "review.sla_breach",
			PayloadJSON: payload, CreatedAt: time.Now().UTC(),
		}).Error
	}
	return out, nil
}

func quoteJoin(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(parts, ",")
}
