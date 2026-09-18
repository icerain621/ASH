package spacepolicy_test

import (
	"testing"
	"time"

	"github.com/ash-repwiki/ash/internal/spacepolicy"
	"github.com/ash-repwiki/ash/internal/store"
)

func TestMergeOrderKindPackScope(t *testing.T) {
	user := spacepolicy.KindDefaults("user")
	if user.MultiSign || user.CitationMode != spacepolicy.CitationOptional {
		t.Fatalf("user defaults: %+v", user)
	}
	team := spacepolicy.KindDefaults("team")
	if !team.MultiSign || team.CitationMode != spacepolicy.CitationRequired {
		t.Fatalf("team defaults: %+v", team)
	}

	// pack softens multiSign=false but MergeStricter keeps true from kind when overlay false?
	// Overlay multiSign=false should NOT weaken: only true wins.
	softened := spacepolicy.MergeStricter(team, spacepolicy.PolicyFragment{
		CitationMode: spacepolicy.CitationOptional, MultiSign: false, ReviewSLAHours: 200,
	})
	if !softened.MultiSign {
		t.Fatal("multiSign must not soften")
	}
	if softened.CitationMode != spacepolicy.CitationRequired {
		t.Fatalf("citation must not soften, got %s", softened.CitationMode)
	}
	if softened.ReviewSLAHours != 72 {
		t.Fatalf("sla must not lengthen, got %d", softened.ReviewSLAHours)
	}

	strict := spacepolicy.MergeStricter(softened, spacepolicy.PolicyFragment{
		CitationMode: spacepolicy.CitationStrict, MultiSign: true, ReviewSLAHours: 24,
	})
	if strict.CitationMode != spacepolicy.CitationStrict || !strict.MultiSign || strict.ReviewSLAHours != 24 {
		t.Fatalf("strict overlay: %+v", strict)
	}
}

func TestGetPutEffectivePolicy(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	now := time.Now().UTC()
	if err := db.Create(&store.Space{
		ID: "sp_team", OrgID: "org1", Name: "Team", Kind: "team", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc := spacepolicy.NewService(db)

	on := true
	sla := 48
	pack, err := svc.PutPack("sp_team", spacepolicy.PutPackRequest{
		CitationMode: spacepolicy.CitationStrict, MultiSign: &on, ReviewSLAHours: &sla,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pack.CitationMode != spacepolicy.CitationStrict || pack.ReviewSLAHours != 48 {
		t.Fatalf("pack=%+v", pack)
	}

	if err := db.Create(&store.ResourceScope{
		ID: "rs1", SpaceID: "sp_team", ResourceType: "memory", ResourceID: "*",
		PolicyJSON: `{"citationMode":"strict","multiSign":true,"reviewSlaHours":12}`,
		CreatedAt:  now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	eff, err := svc.EffectivePolicy("sp_team")
	if err != nil {
		t.Fatal(err)
	}
	if eff.CitationMode != spacepolicy.CitationStrict || !eff.MultiSign || eff.ReviewSLAHours != 12 {
		t.Fatalf("effective=%+v", eff)
	}
	if len(eff.Sources) < 2 {
		t.Fatalf("sources=%v", eff.Sources)
	}
}

func TestParseToolApprovalPresetsFailClosed(t *testing.T) {
	if got := spacepolicy.ParseToolApprovalPresets(""); len(got) != 0 {
		t.Fatalf("empty=%v", got)
	}
	if spacepolicy.PresetAllowsTool(`{}`, "bash") {
		t.Fatal("empty body must not allow")
	}
	if spacepolicy.PresetAllowsTool(`{"toolApprovalPresets":[{"tool":"bash","risk":"high","default":"ask"}]}`, "bash") {
		t.Fatal("ask must not auto-allow")
	}
	if spacepolicy.PresetAllowsTool(`{"toolApprovalPresets":[{"tool":"bash","default":"deny"}]}`, "bash") {
		t.Fatal("deny must not auto-allow")
	}
	if !spacepolicy.PresetAllowsTool(`{"toolApprovalPresets":[{"tool":"bash","risk":"high","default":"allow"}]}`, "bash") {
		t.Fatal("allow preset should auto-allow bash")
	}
	if spacepolicy.PresetAllowsTool(`{"toolApprovalPresets":[{"tool":"bash","default":"allow"}]}`, "other") {
		t.Fatal("other tool must stay fail-closed")
	}
}

func TestPutPackPersistsToolApprovalPresets(t *testing.T) {
	db := store.OpenTest(t, t.TempDir())
	svc := spacepolicy.NewService(db)
	body := `{"toolApprovalPresets":[{"tool":"danger.tool","risk":"high","default":"ask"}]}`
	pack, err := svc.PutPack("local", spacepolicy.PutPackRequest{BodyJSON: body})
	if err != nil {
		t.Fatal(err)
	}
	if pack.BodyJSON != body {
		t.Fatalf("bodyJson=%q", pack.BodyJSON)
	}
	got, err := svc.GetPack("local")
	if err != nil {
		t.Fatal(err)
	}
	presets := spacepolicy.ParseToolApprovalPresets(got.BodyJSON)
	if len(presets) != 1 || presets[0].Tool != "danger.tool" || presets[0].Default != "ask" {
		t.Fatalf("presets=%+v", presets)
	}
	_, err = svc.PutPack("local", spacepolicy.PutPackRequest{BodyJSON: "not-json"})
	if err == nil {
		t.Fatal("expected invalid bodyJson error")
	}
}
