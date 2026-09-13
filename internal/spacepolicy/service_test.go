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
